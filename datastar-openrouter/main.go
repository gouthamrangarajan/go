package main

import (
	"fmt"
	"net/http"

	"datastar-openrouter/services/middlewares"

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
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(middlewares.Authorization)

	router.Get("/", mainPageHandler)
	router.Get("/{sessionId}", mainPageHandler)
	router.Post("/new", newChatHandler)
	router.Post("/chat", promptHandler)
	router.Post("/session/delete", deleteSessionHandler)
	router.Post("/sessions/search", searchSessionHandler)
	router.Post("/fileupload", fileUploadHandler)
	router.Post("/fileupload/remove", removeUploadedFileHandler)
	router.Post("/retry", retryHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
