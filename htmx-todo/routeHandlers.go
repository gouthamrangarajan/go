package main

import (
	"htmx-todo/components"
	"htmx-todo/models"
	"net/http"
	"os"
	"strconv"
)

func MainPage(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	groceries := GetGroceryData(databaseUrl, authToken)
	items, _ := tranformGroceries(groceries)
	components.MainEl(items).Render(r.Context(), w)
}

func AddGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	groceries := GetGroceryData(databaseUrl, authToken)
	items, _ := tranformGroceries(groceries)
	newItem := components.Item{Name: r.FormValue("item"), Quantity: 1, AnimationClass: "animate-slide-down"}
	newItem.Id = InsertGrocery(databaseUrl, authToken, newItem.Name, newItem.Quantity)
	// newItem.Id = rand.Int()
	if newItem.Id != 0 {
		items = append([]components.Item{newItem}, items...)
	}
	components.ItemsUl(items).Render(r.Context(), w)
}
func IncrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		components.ItemQuantityDisplay(id, 0, false).Render(r.Context(), w)
	} else {
		currentQuantity += 1
		rowsAffected := UpdateQuantity(databaseUrl, authToken, id, currentQuantity)
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, 0, false).Render(r.Context(), w)
		}
	}
}
func DecrementGroceryItemQuantity(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, IdErr := strconv.Atoi(r.FormValue("id"))
	currentQuantity, QuantityErr := strconv.Atoi(r.FormValue("currentQuantity"))
	if IdErr != nil || QuantityErr != nil {
		components.ItemQuantityDisplay(id, 0, false).Render(r.Context(), w)
	} else {
		currentQuantity -= 1
		if currentQuantity < 1 {
			currentQuantity = 1
		}
		rowsAffected := UpdateQuantity(databaseUrl, authToken, id, currentQuantity)
		if rowsAffected != 0 {
			components.ItemQuantityDisplay(id, currentQuantity, true).Render(r.Context(), w)
		} else {
			components.ItemQuantityDisplay(id, 0, false).Render(r.Context(), w)
		}
	}
}
func DeleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	groceries := GetGroceryData(databaseUrl, authToken)
	items, idToIndexMap := tranformGroceries(groceries)
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		rowsAffected := DeleteGrocery(databaseUrl, authToken, id)
		if rowsAffected != 0 {
			items = findAndRemoveItem(items, id, idToIndexMap)
		}
	}
	components.ItemsUl(items).Render(r.Context(), w)
}
func ToggleCompleteGroceryItem(w http.ResponseWriter, r *http.Request) {
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	databaseUrl := os.Getenv("TURSO_DATABASE_URL")
	id, err := strconv.Atoi(r.FormValue("id"))
	if err == nil {
		groceryModelItem := GetGroceryItemData(databaseUrl, authToken, id)
		item := transformGrocery(groceryModelItem)
		rowsAffected := UpdateCompleted(databaseUrl, authToken, id, !groceryModelItem.Completed)
		if rowsAffected != 0 {
			groceryModelItem.Completed = !groceryModelItem.Completed
			item.Completed = !item.Completed
		}
		components.ItemNameDisplay(item).Render(r.Context(), w)
	}
}
func findAndRemoveItem(items []components.Item, id int, idToIndexMap map[int]int) []components.Item {
	if idToIndexMap != nil {
		index, ok := idToIndexMap[id]
		if ok {
			items = append(items[:index], items[index+1:]...)
			idToIndexMap = nil
			return items
		}
	}
	for i, item := range items {
		if item.Id == id {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}

	return items
}

func tranformGroceries(list []models.Grocery) ([]components.Item, map[int]int) {
	items := []components.Item{}
	itemIdToIndexMap := make(map[int]int)
	for _, grocery := range list {
		item := components.Item{Id: grocery.Id, Name: grocery.Description, Quantity: grocery.Quantity, Completed: grocery.Completed, AnimationClass: ""}
		items = append(items, item)
		itemIdToIndexMap[grocery.Id] = len(items) - 1
	}
	return items, itemIdToIndexMap
}

func transformGrocery(grocery models.Grocery) components.Item {
	return components.Item{Id: grocery.Id, Name: grocery.Description, Quantity: grocery.Quantity, Completed: grocery.Completed, AnimationClass: ""}
}
