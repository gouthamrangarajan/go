package main

import (
	"fmt"
	"htmx-todo/components"
	"htmx-todo/models"
	"htmx-todo/services"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func MainPage(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	suggestions := r.URL.Query().Get("suggestions")
	valid := services.ValidateUserIdInCookie(r)
	if !valid {
		components.MainElForLogin(sort).Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceries := <-*services.GetGroceryListViaChannel(databaseUrl, authToken, sort)
		items, _ := tranformGroceryList(groceries, false)
		components.MainEl(items, sort, suggestions).Render(r.Context(), w)
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	hashedTokenFromConfig := os.Getenv("TOKEN")
	token := r.FormValue("token")
	sort := r.FormValue("sort")
	compareErr := bcrypt.CompareHashAndPassword([]byte(hashedTokenFromConfig), []byte(token))
	if compareErr != nil {
		components.LoginFormErrMsg().Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceriesChannel := services.GetGroceryListViaChannel(databaseUrl, authToken, sort)
		cookie := services.GenerateUserIdCookie()
		http.SetCookie(w, &cookie)
		groceries := <-*groceriesChannel
		items, _ := tranformGroceryList(groceries, true)
		components.SectionEl(items, sort, true, "").Render(r.Context(), w)
	}
}

func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	sort := r.FormValue("sort")

	openaiChannel := callOpenAI(r.FormValue("item"))
	newItemId := <-*services.InsertGroceryItemViaChannel(databaseUrl, authToken, r.FormValue("item"), 1)
	groceries := <-*services.GetGroceryListViaChannel(databaseUrl, authToken, sort)
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
	sort := r.FormValue("sort")
	suggestions := r.FormValue("suggestions")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		<-*services.DeleteGroceryItemViaChannel(databaseUrl, authToken, id)
	}
	groceries := <-*services.GetGroceryListViaChannel(databaseUrl, authToken, sort)
	items, _ := tranformGroceryList(groceries, false)
	components.OpenAiSuggestionsAndItemsUl(items, suggestions, true).Render(r.Context(), w)
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
		rowsAffected := <-*services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity)
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
		rowsAffected := <-*services.UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity)
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
		groceryModelItem := <-*services.GetGroceryListItemViaChannel(databaseUrl, authToken, id)
		item := transformGrocery(groceryModelItem)
		rowsAffected := <-*services.UpdateCompletedFieldGroceryItemViaChannel(databaseUrl, authToken, id, !groceryModelItem.Completed)
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

func callOpenAI(item string) <-chan string {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")
	noOfItemsToSuggest := os.Getenv("OPENAI_API_NUMBER_OF_ITEMS_TO_SUGGEST")

	prompt := fmt.Sprintf("Give %v items for grocery similar to %v in the format: 'item 1, item 2, item 3'", noOfItemsToSuggest, item)

	request := services.OpenAIAPIRequest{
		Model: model,
		Messages: append([]services.OpenAIAPIRequestMessageField{},
			services.OpenAIAPIRequestMessageField{Role: "user", Content: prompt}),
	}
	return services.CallOpenAIViaChannel(url, key, request)
}
