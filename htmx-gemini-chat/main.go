package main

import (
	"fmt"
	"htmx-gemini-chat/services"
	"net/http"
	"time"

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
	http.HandleFunc("/new", services.CookieHandlerToNewChatSession)
	http.HandleFunc("/send", services.CookieHandlerToPromptHandler)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	fmt.Println("Listening on :3000")
	server := &http.Server{
		Addr:         ":3000",
		ReadTimeout:  60 * time.Second, // Increase read timeout
		WriteTimeout: 60 * time.Second, // Increase write timeout
	}
	server.ListenAndServe()
}
