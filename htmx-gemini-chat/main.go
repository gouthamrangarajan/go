package main

import (
	"fmt"
	"htmx-gemini-chat/services"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Success loaded .env file")
	}
	router := chi.NewRouter()
	router.Get("/", func(response http.ResponseWriter, request *http.Request) {
		services.CookieHandlerToMainPage(response, request, 0)
	})
	router.Get("/{sessionId}", func(response http.ResponseWriter, request *http.Request) {
		sessionIdStr := chi.URLParam(request, "sessionId")
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			sessionId = -1
		}
		services.CookieHandlerToMainPage(response, request, sessionId)
	})
	router.Post("/new", services.CookieHandlerToNewChatSession)
	router.Post("/send", services.CookieHandlerToPromptHandler)
	router.Get("/{sessionId}/{conversationId}", func(response http.ResponseWriter, request *http.Request) {
		sessionIdStr := chi.URLParam(request, "sessionId")
		conversationIdStr := chi.URLParam(request, "conversationId")
		sessionId, _ := strconv.Atoi(sessionIdStr)
		conversationId, _ := strconv.Atoi(conversationIdStr)
		services.CookieHandlerToChatConversation(response, request, sessionId, conversationId)
	})

	router.Get("/assets/*", func(response http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(response, request)
	})

	fmt.Println("Listening on :3000")
	server := &http.Server{
		Addr:         ":3000",
		ReadTimeout:  60 * time.Second, // Increase read timeout
		WriteTimeout: 60 * time.Second, // Increase write timeout
		Handler:      router,
	}
	server.ListenAndServe()
}
