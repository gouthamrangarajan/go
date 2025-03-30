package main

import (
	"fmt"
	"htmx-calendar/services"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	http.Handle("/", services.MiddlewareUI(MainPage))
	http.Handle("/add", services.MiddlewareUI(Add))
	http.HandleFunc("/login", Login)
	http.Handle("/dnd", services.MiddlewareJSON(UpdateDate))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
