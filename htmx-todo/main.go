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
	}

	http.HandleFunc("/", MainPage)
	http.HandleFunc("/add", AddGroceryItem)
	http.HandleFunc("/delete", DeleteGroceryItem)
	http.HandleFunc("/increment", IncrementGroceryItemQuantity)
	http.HandleFunc("/decrement", DecrementGroceryItemQuantity)
	http.HandleFunc("/complete", ToggleCompleteGroceryItem)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
