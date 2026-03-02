package main

import (
	"datastar-openrouter/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
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
	promptRouter := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(services.AuthorizationMiddleware)

	rateLimitSecondsStr := os.Getenv("RATE_LIMIT_SECONDS")
	rateLimitRequestsStr := os.Getenv("RATE_LIMIT_REQUESTS")

	rateLimitSeconds, err := strconv.Atoi(rateLimitSecondsStr)
	if err != nil {
		rateLimitSeconds = 5
	}
	rateLimitRequests, err := strconv.Atoi(rateLimitRequestsStr)
	if err != nil {
		rateLimitRequests = 10
	}
	promptRouter.Use(httprate.LimitByIP(rateLimitRequests, time.Duration(rateLimitSeconds)*time.Second)) // 10 request in 5 seconds

	router.Get("/", mainPageHandler)
	router.Get("/{sessionId}", mainPageHandler)
	router.Post("/conversations", convertConversationsMarkdownHandler)
	router.Post("/new", newChatHandler)
	promptRouter.Post("/chat", promptHandler)
	router.Post("/session/delete", deleteSessionHandler)
	router.Post("/sessions/search", searchSessionHandler)
	router.Post("/fileupload", fileUploadHandler)
	router.Post("/fileupload/remove", removeUploadedFileHandler)
	router.Post("/retry", retryHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Mount("/", promptRouter)
	http.ListenAndServe(":3000", router)
}
