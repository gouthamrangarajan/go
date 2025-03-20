package main

import (
	"htmx-todo/models"
)

func GetGroceryListViaChannel(databaseUrl string, authToken string, sort string) *chan []models.Grocery {
	channel := make(chan []models.Grocery)
	go func() {
		defer close(channel)
		channel <- GetGroceryList(databaseUrl, authToken, sort)
	}()
	return &channel
}

func GetGroceryListItemViaChannel(databaseUrl string, authToken string, id int) *chan models.Grocery {
	channel := make(chan models.Grocery)
	go func() {
		defer close(channel)
		channel <- GetGroceryListItem(databaseUrl, authToken, id)
	}()
	return &channel
}

func InsertGroceryItemViaChannel(databaseUrl string, authToken string, name string, quantity int) *chan int {
	channel := make(chan int)
	go func() {
		defer close(channel)
		channel <- InsertGroceryItem(databaseUrl, authToken, name, quantity)
	}()
	return &channel
}

func DeleteGroceryItemViaChannel(databaseUrl string, authToken string, id int) *chan int {
	channel := make(chan int)
	go func() {
		defer close(channel)
		channel <- DeleteGroceryItem(databaseUrl, authToken, id)
	}()
	return &channel
}

func UpdateQuantityGroceryItemViaChannel(databaseUrl string, authToken string, id int, quantity int) *chan int {
	channel := make(chan int)
	go func() {
		defer close(channel)
		channel <- UpdateQuantityGroceryItem(databaseUrl, authToken, id, quantity)
	}()
	return &channel
}

func UpdateCompletedFieldGroceryItemViaChannel(databaseUrl string, authToken string, id int, completed bool) *chan int {
	channel := make(chan int)
	go func() {
		defer close(channel)
		channel <- UpdateCompletedFieldGroceryItem(databaseUrl, authToken, id, completed)
	}()
	return &channel
}
