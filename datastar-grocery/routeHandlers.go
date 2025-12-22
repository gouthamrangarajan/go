package main

import (
	"datastar-grocery/components"
	"datastar-grocery/models"
	"datastar-grocery/services"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/crypto/bcrypt"
)

var changeSignalAndSortValueChannels = make(map[string]chan string)
var sort = ""

func MainPageWithChi(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	suggestions := r.URL.Query().Get("suggestions")
	components.MainEl(os.Getenv("LOCATION"), sort, suggestions).Render(r.Context(), w)
}

func Login(w http.ResponseWriter, r *http.Request) {
	hashedTokenFromConfig := os.Getenv("TOKEN")
	token := r.FormValue("token")
	sort := r.FormValue("sort")
	suggestions := r.FormValue("suggestions")
	compareErr := bcrypt.CompareHashAndPassword([]byte(hashedTokenFromConfig), []byte(token))
	if compareErr != nil {
		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(components.LoginFormErrMsg(), datastar.WithUseViewTransitions(true))
		return
	}
	cookie := services.GenerateUserIdCookie()
	http.SetCookie(w, &cookie)
	sse := datastar.NewSSE(w, r)
	sse.PatchElementTempl(components.SectionEl(os.Getenv("LOCATION"), sort, suggestions), datastar.WithUseViewTransitions(true))
}
func GroceryItemList(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sort := r.URL.Query().Get("sort")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	groceries := services.GetGroceryList(databaseUrl, authToken, sort)
	items, _ := tranformGroceryList(groceries, false)
	sse.PatchElementTempl(components.ItemsUL(items), datastar.WithUseViewTransitions(true))
	ipAddress := r.RemoteAddr
	randomInt := rand.Intn(1000)
	// fmt.Println("New SSE connection from ", ipAddress, " with key ", randomInt)
	key := fmt.Sprintf("%v-%v", ipAddress, randomInt)
	changeSignalAndSortValueChannels[key] = make(chan string)
	for {
		select {
		case <-r.Context().Done():
			channelToClose := changeSignalAndSortValueChannels[key]
			defer close(channelToClose)
			delete(changeSignalAndSortValueChannels, key)
			return

		case sortVal := <-changeSignalAndSortValueChannels[key]:
			groceries := services.GetGroceryList(databaseUrl, authToken, sortVal)
			items, _ := tranformGroceryList(groceries, false)
			sse.PatchElementTempl(components.ItemsUL(items), datastar.WithUseViewTransitions(true))
		}
	}
}
func GroceryItemListChangeSort(w http.ResponseWriter, r *http.Request) {
	sort = r.URL.Query().Get("sort")
	sendSignalToChangeChannels()
}
func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	newItem := strings.Trim(r.FormValue("item"), " ")

	openaiChannel := make(chan string)
	defer close(openaiChannel)
	callOpenAI(newItem, openaiChannel)
	newItemChannel := make(chan int)
	defer close(newItemChannel)
	go services.InsertGroceryItemViaChannel(databaseUrl, authToken, newItem, 1, newItemChannel)
	sse := datastar.NewSSE(w, r)
	openAiResult := <-openaiChannel
	sse.PatchElementTempl(components.OpenAiSuggestions(openAiResult), datastar.WithUseViewTransitions(true))

	if strings.Contains(newItem, " ") {
		newItem = strings.Split(newItem, " ")[0]
		sse.PatchSignals([]byte("{_newItem:'" + newItem + "'}"))
	} else {
		sse.PatchSignals([]byte("{_newItem:''}"))
	}
	rowsAffected := <-newItemChannel
	if rowsAffected != 0 {
		sendSignalToChangeChannels()
	}
}
func RemoveGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		channel := make(chan int)
		defer close(channel)
		go services.DeleteGroceryItemViaChannel(databaseUrl, authToken, id, channel)
		rowsAffected := <-channel
		if rowsAffected != 0 {
			sendSignalToChangeChannels()
		}
	}
	//handle errror
	w.WriteHeader(http.StatusOK)
}
func IncrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		return
	}
	currentQuantity += 1
	channel := make(chan int)
	defer close(channel)
	go services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity, channel)
	rowsAffected := <-channel
	if rowsAffected != 0 {
		sendSignalToChangeChannels()
	}

}
func DecrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		return
	}
	currentQuantity -= 1
	if currentQuantity < 1 {
		currentQuantity = 1
	}
	channel := make(chan int)
	defer close(channel)
	go services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity, channel)
	rowsAffected := <-channel
	if rowsAffected != 0 {
		sendSignalToChangeChannels()
	}

}

func ToggleCompleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		return
	}

	groceryItemChannel := make(chan models.Grocery)
	defer close(groceryItemChannel)
	go services.GetGroceryListItemViaChannel(databaseUrl, authToken, id, groceryItemChannel)
	groceryModelItem := <-groceryItemChannel
	item := transformGrocery(groceryModelItem)
	updateChannel := make(chan int)
	defer close(updateChannel)
	go services.UpdateCompletedFieldGroceryItemViaChannel(databaseUrl, authToken, id, !groceryModelItem.Completed, updateChannel)
	rowsAffected := <-updateChannel
	if rowsAffected != 0 {
		groceryModelItem.Completed = !groceryModelItem.Completed
		item.Completed = !item.Completed
		sendSignalToChangeChannels()
	}

}

func tranformGroceryList(list []models.Grocery, animateAllItems bool) ([]components.Item, map[int]int) {
	items := []components.Item{}
	itemIdToIndexMap := make(map[int]int)
	for _, grocery := range list {
		item := transformGrocery(grocery)
		if animateAllItems {
			item.AnimationClass = "animate-slide-down"
		}
		items = append(items, item)
		itemIdToIndexMap[grocery.Id] = len(items) - 1
	}
	return items, itemIdToIndexMap
}

func transformGrocery(grocery models.Grocery) components.Item {
	return components.Item{Id: grocery.Id, Name: grocery.Description, Quantity: grocery.Quantity, Completed: grocery.Completed, AnimationClass: ""}
}

func callOpenAI(item string, channel chan<- string) {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")
	noOfItemsToSuggest := os.Getenv("OPENAI_API_NUMBER_OF_ITEMS_TO_SUGGEST")

	prompt := fmt.Sprintf("Give %v items for grocery similar to %v in the format: 'item 1, item 2, item 3'. Do not give me the same item in different variations.If the item provided is not grocery do not suggest any items.", noOfItemsToSuggest, item)

	request := services.OpenAIAPIRequest{
		Model: model,
		Messages: append([]services.OpenAIAPIRequestMessageField{},
			services.OpenAIAPIRequestMessageField{Role: "user", Content: prompt}),
	}
	go services.CallOpenAIViaChannel(url, key, request, channel)
}

func sendSignalToChangeChannels() {
	for _, changeChannel := range changeSignalAndSortValueChannels {
		changeChannel <- sort
	}
}
