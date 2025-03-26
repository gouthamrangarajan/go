package main

import (
	"context"
	"fmt"
	"htmx-grocery/components"
	"htmx-grocery/models"
	"htmx-grocery/services"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func MainPage(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	suggestions := r.URL.Query().Get("suggestions")
	valid := services.ValidateUserIdInCookie(r)
	if !valid {
		components.MainElForLogin(sort, suggestions).Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceriesChannel := make(chan []models.Grocery)
		defer close(groceriesChannel)
		go services.GetGroceryListViaChannel(databaseUrl, authToken, sort, groceriesChannel)
		groceries := <-groceriesChannel
		items, _ := tranformGroceryList(groceries, false)
		components.MainEl(items, sort, suggestions).Render(r.Context(), w)
		// comp := components.MainEl(items, sort, suggestions)
		// templ.Handler(comp).ServeHTTP(w, r)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	hashedTokenFromConfig := os.Getenv("TOKEN")
	token := r.FormValue("token")
	sort := r.FormValue("sort")
	suggestions := r.FormValue("suggestions")
	compareErr := bcrypt.CompareHashAndPassword([]byte(hashedTokenFromConfig), []byte(token))
	if compareErr != nil {
		components.LoginFormErrMsg().Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceriesChannel := make(chan []models.Grocery)
		defer close(groceriesChannel)
		go services.GetGroceryListViaChannel(databaseUrl, authToken, sort, groceriesChannel)
		cookie := services.GenerateUserIdCookie()
		http.SetCookie(w, &cookie)
		groceries := <-groceriesChannel
		items, _ := tranformGroceryList(groceries, true)
		components.SectionEl(items, sort, true, suggestions).Render(r.Context(), w)
	}
}

func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	sort := r.FormValue("sort")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	openaiChannel := make(chan string)
	defer close(openaiChannel)
	callOpenAI(r.FormValue("item"), openaiChannel, ctx)
	newItemChannel := make(chan int)
	defer close(newItemChannel)
	go services.InsertGroceryItemViaChannel(databaseUrl, authToken, r.FormValue("item"), 1, newItemChannel)
	newItemId := <-newItemChannel
	groceriesChannel := make(chan []models.Grocery)
	defer close(groceriesChannel)
	go services.GetGroceryListViaChannel(databaseUrl, authToken, sort, groceriesChannel)
	groceries := <-groceriesChannel
	items, idToIndexMap := tranformGroceryList(groceries, false)
	if newItemId != 0 {
		items[idToIndexMap[newItemId]].AnimationClass = "animate-slide-down"
	}
	openAiResult := <-openaiChannel
	components.OpenAiSuggestionsAndItemsUl(items, openAiResult, true).Render(r.Context(), w)
}
func RemoveGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		channel := make(chan int)
		defer close(channel)
		go services.DeleteGroceryItemViaChannel(databaseUrl, authToken, id, channel)
		<-channel
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
		components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
	} else {
		currentQuantity += 1
		channel := make(chan int)
		defer close(channel)
		go services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity, channel)
		rowsAffected := <-channel
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
		}
	}
}
func DecrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
	} else {
		currentQuantity -= 1
		if currentQuantity < 1 {
			currentQuantity = 1
		}
		channel := make(chan int)
		defer close(channel)
		go services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity, channel)
		rowsAffected := <-channel
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, currentQuantity, false).Render(r.Context(), w)
		}
	}
}

func ToggleCompleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
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
		}
		components.ItemNameDisplay(item).Render(r.Context(), w)
	} else {
		components.ItemNameDisplay(components.Item{}).Render(r.Context(), w)
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

func callOpenAI(item string, channel chan<- string, ctx context.Context) {
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
	go services.CallOpenAIViaChannelTillContext(url, key, request, channel, ctx)
}
