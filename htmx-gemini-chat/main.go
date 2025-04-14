package main

import (
	"fmt"
	"htmx-gemini-chat/services"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Success loaded .env file")
	}
	http.HandleFunc("/", services.CookieHandlerToMainPage)
	http.HandleFunc("/send", services.CookieHandlerToPromptHandler)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
