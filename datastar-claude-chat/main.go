package main

import (
	"datastar-claude-chat/services/middlewares"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Use(middlewares.Authorization)

	router.Get("/", mainPageHandler)
	router.Get("/{sessionId}", mainPageHandler)
	router.Get("/sessions/search", menuSearchHandler)
	router.Post("/new", newChatHandler)
	router.Post("/delete", deleteChatHandler)
	router.Post("/prompt", promptHandler)
	router.Post("/fileupload", fileuploadHandler)
	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
