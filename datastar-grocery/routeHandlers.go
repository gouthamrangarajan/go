package main

import (
	"bytes"
	"context"
	"datastar-grocery/components"
	"datastar-grocery/models"
	"datastar-grocery/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/crypto/bcrypt"
)

var changeSignalMap sync.Map
var sort atomic.Value

func MainPageWithChi(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	suggestions := r.URL.Query().Get("suggestions")
	model := models.MainElData{Location: os.Getenv("LOCATION"), Sort: sort, Suggestions: suggestions, SId: uuid.NewString()}
	changeSignalMap.Store(model.SId, make(chan models.LongSSEChannelData))
	components.MainEl(model).Render(r.Context(), w)
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
	cookie, err := services.GenerateUserIdCookie()
	if err == nil {
		http.SetCookie(w, &cookie)
	}
	sse := datastar.NewSSE(w, r)
	model := models.MainElData{Location: os.Getenv("LOCATION"), Sort: sort, Suggestions: suggestions, SId: uuid.NewString()}
	changeSignalMap.Store(model.SId, make(chan models.LongSSEChannelData))
	sse.PatchElementTempl(components.SectionEl(model), datastar.WithUseViewTransitions(true))
}
func GroceryItemList(w http.ResponseWriter, r *http.Request) {
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(r, &ClientSignals)
	// fmt.Printf("ClientSignals: %v\n", ClientSignals)
	sse := datastar.NewSSE(w, r)
	// sort := r.URL.Query().Get("sort")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	groceries := services.GetGroceryList(databaseUrl, authToken, getSort())
	items, _ := tranformGroceryList(groceries, false)
	sse.PatchElementTempl(components.ItemsUL(items), datastar.WithUseViewTransitions(true))
	sse.PatchSignals([]byte("{_loadingItems:false}"))

	session, exists := changeSignalMap.Load(ClientSignals.SId)
	if !exists {
		session = make(chan string)
		changeSignalMap.Store(ClientSignals.SId, session)
	}

	for {
		select {
		case <-r.Context().Done():
			defer close(session.(chan models.LongSSEChannelData))
			changeSignalMap.Delete(ClientSignals.SId)
			return

		case data := <-session.(chan models.LongSSEChannelData):
			if data.IsSignal {
				sse.PatchSignals([]byte(data.Content))
			} else if data.FullRefresh {
				groceries := services.GetGroceryList(databaseUrl, authToken, data.SortVal)
				items, _ := tranformGroceryList(groceries, false)
				sse.PatchElementTempl(components.ItemsUL(items), datastar.WithUseViewTransitions(true))
			} else {
				sse.PatchElements(data.Content, datastar.WithUseViewTransitions(true))
			}
		}
	}
}
func GroceryItemListChangeSort(w http.ResponseWriter, r *http.Request) {
	sortVal := r.URL.Query().Get("sort")
	setSort(sortVal)
	sendSignalToChangeChannels(models.LongSSEChannelData{FullRefresh: true, SortVal: sortVal})
}
func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	newItem := strings.Trim(r.FormValue("item"), " ")

	aiChannel := make(chan string)
	defer close(aiChannel)
	callOpenRouter(newItem, aiChannel)
	newItemChannel := make(chan int)
	defer close(newItemChannel)
	go services.InsertGroceryItemViaChannel(databaseUrl, authToken, newItem, 1, newItemChannel)
	rowsAffected := <-newItemChannel
	if rowsAffected != 0 {
		sendSignalToChangeChannels(models.LongSSEChannelData{FullRefresh: true, SortVal: getSort()})
	}
	if strings.Contains(newItem, " ") {
		newItem = strings.Split(newItem, " ")[0]
		sendSignalToChangeChannels(models.LongSSEChannelData{Content: "{_newItem:'" + newItem + "'}", IsSignal: true})
	} else {
		sendSignalToChangeChannels(models.LongSSEChannelData{Content: "{_newItem:''}", IsSignal: true})
	}

	aiResult := <-aiChannel
	aiSuggestionBuffer := new(bytes.Buffer)
	components.AiSuggestions(aiResult).Render(context.Background(), aiSuggestionBuffer)
	sendSignalToChangeChannels(models.LongSSEChannelData{Content: aiSuggestionBuffer.String()})

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
			sendSignalToChangeChannels(models.LongSSEChannelData{FullRefresh: true, SortVal: getSort()})
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
		displayBuff := new(bytes.Buffer)
		components.ItemQuantityDisplay(id, currentQuantity).Render(context.Background(), displayBuff)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: displayBuff.String(),
		})
		inputBuff1 := new(bytes.Buffer)
		components.IncreaseQuantityFormInput(id, currentQuantity).Render(context.Background(), inputBuff1)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: inputBuff1.String(),
		})
		inputBuff2 := new(bytes.Buffer)
		components.DecreaseQuantityFormInput(id, currentQuantity).Render(context.Background(), inputBuff2)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: inputBuff2.String(),
		})
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
		displayBuff := new(bytes.Buffer)
		components.ItemQuantityDisplay(id, currentQuantity).Render(context.Background(), displayBuff)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: displayBuff.String(),
			SortVal: getSort(),
		})
		inputBuff1 := new(bytes.Buffer)
		components.DecreaseQuantityFormInput(id, currentQuantity).Render(context.Background(), inputBuff1)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: inputBuff1.String(),
		})
		inputBuff2 := new(bytes.Buffer)
		components.IncreaseQuantityFormInput(id, currentQuantity).Render(context.Background(), inputBuff2)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: inputBuff2.String(),
		})
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
		buff := new(bytes.Buffer)
		components.ItemNameDisplay(item).Render(context.Background(), buff)
		sendSignalToChangeChannels(models.LongSSEChannelData{
			Content: buff.String(),
			SortVal: getSort(),
		})
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
func callOpenRouter(item string, channel chan<- string) {
	model := os.Getenv("OPENROUTER_API_MODEL")
	noOfItemsToSuggest := os.Getenv("OPENROUTER_API_NUMBER_OF_ITEMS_TO_SUGGEST")

	prompt := fmt.Sprintf("Give %v items for grocery similar to %v in the format: 'item 1, item 2, item 3'. Do not give me the same item in different variations.If the item provided is not grocery do not suggest any items.", noOfItemsToSuggest, item)

	request := models.OpenRouterRequest{
		Model: model,
		Messages: append([]models.OpenRouterRequestMessage{},
			models.OpenRouterRequestMessage{Role: "user", Content: prompt}),
	}
	go services.CallOpenRouter(request, channel)
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

func sendSignalToChangeChannels(data models.LongSSEChannelData) {
	changeSignalMap.Range(func(key, value any) bool {
		channel := value.(chan models.LongSSEChannelData)
		channel <- data
		return true
	})
}

func setSort(val string) {
	sort.Store(val)
}

func getSort() string {
	if sort.Load() == nil {
		return ""
	}
	return sort.Load().(string)
}
