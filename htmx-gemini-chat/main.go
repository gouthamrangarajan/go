package main

import (
	"fmt"
	"htmx-gemini-chat/services"
	"htmx-gemini-chat/services/middlewares"
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
	router.Use(middlewares.Authorization)
	router.Get("/", func(response http.ResponseWriter, request *http.Request) {
		services.MainPageHandler(response, request, 0)
	})
	router.Get("/{sessionId}", func(response http.ResponseWriter, request *http.Request) {
		sessionIdStr := chi.URLParam(request, "sessionId")
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			sessionId = -1
		}
		services.MainPageHandler(response, request, sessionId)
	})
	router.Get("/{sessionId}/{conversationId}", func(response http.ResponseWriter, request *http.Request) {
		sessionIdStr := chi.URLParam(request, "sessionId")
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			sessionId = -1
		}
		conversationIdStr := chi.URLParam(request, "conversationId")
		conversationId, err := strconv.Atoi(conversationIdStr)
		if err != nil {
			conversationId = -1
		}
		if sessionId == -1 || conversationId == -1 {
			http.Error(response, "Invalid request", http.StatusBadRequest)
			return
		}
		services.MarkdownSrcHandler(sessionId, conversationId, response, request)
	})
	router.Post("/new", services.NewChatSessionHandler)
	router.Post("/send", services.PromptHandler)
	router.Delete("/delete/{sessionId}", func(response http.ResponseWriter, request *http.Request) {
		sessionIdStr := chi.URLParam(request, "sessionId")
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			sessionId = -1
		}
		services.DeleteSessionHandler(response, request, sessionId)
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
