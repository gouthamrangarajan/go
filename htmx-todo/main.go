package main

import (
	"fmt"
	"net/http"

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
	http.Handle("/add", Middleware(AddGroceryItem))
	http.Handle("/delete", Middleware(DeleteGroceryItem))
	http.Handle("/increment", Middleware(IncrementGroceryItemQuantity))
	http.Handle("/decrement", Middleware(DecrementGroceryItemQuantity))
	http.Handle("/complete", Middleware(ToggleCompleteGroceryItem))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
