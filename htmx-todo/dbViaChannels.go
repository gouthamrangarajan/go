package main

import (
	"htmx-todo/models"
	"sync"
)

func GetGroceryListViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, sort string) *chan []models.Grocery {
	wg.Add(1)
	channel := make(chan []models.Grocery)
	go func() {
		defer wg.Done()
		channel <- GetGroceryList(databaseUrl, authToken, sort)
		close(channel)
	}()
	return &channel
}

func GetGroceryListItemViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, id int) *chan models.Grocery {
	wg.Add(1)
	channel := make(chan models.Grocery)
	go func() {
		defer wg.Done()
		channel <- GetGroceryListItem(databaseUrl, authToken, id)
		close(channel)
	}()
	return &channel
}

func InsertGroceryItemViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, name string, quantity int) *chan int {
	wg.Add(1)
	channel := make(chan int)
	go func() {
		defer wg.Done()
		channel <- InsertGroceryItem(databaseUrl, authToken, name, quantity)
		close(channel)
	}()
	return &channel
}

func DeleteGroceryItemViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, id int) *chan int {
	wg.Add(1)
	channel := make(chan int)
	go func() {
		defer wg.Done()
		channel <- DeleteGroceryItem(databaseUrl, authToken, id)
		close(channel)
	}()
	return &channel
}

func UpdateQuantityGroceryItemViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, id int, quantity int) *chan int {
	wg.Add(1)
	channel := make(chan int)
	go func() {
		defer wg.Done()
		channel <- UpdateQuantityGroceryItem(databaseUrl, authToken, id, quantity)
		close(channel)
	}()
	return &channel
}

func UpdateCompletedFieldGroceryItemViaChannel(wg *sync.WaitGroup, databaseUrl string, authToken string, id int, completed bool) *chan int {
	wg.Add(1)
	channel := make(chan int)
	go func() {
		defer wg.Done()
		channel <- UpdateCompletedFieldGroceryItem(databaseUrl, authToken, id, completed)
		close(channel)
	}()
	return &channel
}
