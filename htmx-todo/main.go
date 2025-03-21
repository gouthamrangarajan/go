package main

import (
	"fmt"
	"net/http"

	"htmx-todo/services"

	"github.com/lpernett/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	http.HandleFunc("/", MainPage)
	http.HandleFunc("/login", Login)
	http.Handle("/add", services.Middleware(AddGroceryItem))
	http.Handle("/delete", services.Middleware(RemoveGroceryItem))
	http.Handle("/increment", services.Middleware(IncrementGroceryItemQuantity))
	http.Handle("/decrement", services.Middleware(DecrementGroceryItemQuantity))
	http.Handle("/complete", services.Middleware(ToggleCompleteGroceryItem))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
