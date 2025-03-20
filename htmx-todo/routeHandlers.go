package main

import (
	"htmx-todo/components"
	"htmx-todo/models"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func MainPage(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	valid := ValidateUserIdInCookie(r)
	if !valid {
		components.MainElForLogin(sort).Render(r.Context(), w)
	} else {
		authToken := os.Getenv("TURSO_AUTH_TOKEN")
		databaseUrl := os.Getenv("TURSO_DATABASE_URL")
		groceries := <-*GetGroceryListViaChannel(databaseUrl, authToken, sort)
		// groceries := GetGroceryData(databaseUrl, authToken, sort)
		items, _ := tranformGroceryList(groceries, false)
		components.MainEl(items, sort).Render(r.Context(), w)
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
		groceriesChannel := GetGroceryListViaChannel(databaseUrl, authToken, sort)
		cookie := GenerateUserIdCookie()
		http.SetCookie(w, &cookie)
		groceries := <-*groceriesChannel
		// groceries := GetGroceryList(databaseUrl, authToken, sort)
		items, _ := tranformGroceryList(groceries, true)
		components.SectionEl(items, sort, true).Render(r.Context(), w)
	}
}

func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	sort := r.FormValue("sort")

	// newItem := components.Item{Name: r.FormValue("item"), Quantity: 1, AnimationClass: "animate-slide-down"}
	newItemId := <-*InsertGroceryItemViaChannel(databaseUrl, authToken, r.FormValue("item"), 1)
	// newItem.Id = InsertGroceryItem(databaseUrl, authToken, newItem.Name, newItem.Quantity)
	// groceries := GetGroceryList(databaseUrl, authToken, sort)
	groceries := <-*GetGroceryListViaChannel(databaseUrl, authToken, sort)
	items, idToIndexMap := tranformGroceryList(groceries, false)
	if newItemId != 0 {
		items[idToIndexMap[newItemId]].AnimationClass = "animate-slide-down"
	}
	components.ItemsUl(items).Render(r.Context(), w)
}
func RemoveGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	sort := r.FormValue("sort")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		<-*DeleteGroceryItemViaChannel(databaseUrl, authToken, id)
	}
	groceries := <-*GetGroceryListViaChannel(databaseUrl, authToken, sort)
	items, _ := tranformGroceryList(groceries, false)
	// groceries := GetGroceryList(databaseUrl, authToken, sort)

	// id, err := strconv.Atoi(r.FormValue("id"))
	// if err == nil {
	// 	rowsAffected := DeleteGroceryItem(databaseUrl, authToken, id)
	// 	if rowsAffected != 0 {
	// 		items = findAndRemoveItem(items, id, idToIndexMap)
	// 	}
	// }
	components.ItemsUl(items).Render(r.Context(), w)
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
		rowsAffected := <-*UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity)
		// rowsAffected := UpdateQuantityGroceryItem(databaseUrl, authToken, id, currentQuantity)
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
		rowsAffected := <-*UpdateQuantityGroceryItemViaChannel(databaseUrl, authToken, id, currentQuantity)
		// rowsAffected := UpdateQuantityGroceryItem(databaseUrl, authToken, id, currentQuantity)
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
		groceryModelItem := <-*GetGroceryListItemViaChannel(databaseUrl, authToken, id)
		item := transformGrocery(groceryModelItem)
		rowsAffected := <-*UpdateCompletedFieldGroceryItemViaChannel(databaseUrl, authToken, id, !groceryModelItem.Completed)
		// groceryModelItem := GetGroceryListItem(databaseUrl, authToken, id)
		// item := transformGrocery(groceryModelItem)
		// rowsAffected := UpdateCompletedFieldGroceryItem(databaseUrl, authToken, id, !groceryModelItem.Completed)
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
