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
	services.InitDB()
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
	promptRouter.Use(middleware.ClientIPFromXFFTrustedProxies(1))
	promptRouter.Use(httprate.LimitBy(
		rateLimitRequests,
		time.Duration(rateLimitSeconds)*time.Second,	
		func(request *http.Request) (string, error) {
			// Get the IP that middleware.RealIP has already verified
			ip := middleware.GetClientIP(request.Context())
			// Canonicalize handles IPv6 /64 subnets naturally
			// Fallback: If for some reason the middleware failed (local dev), 
			// use the direct network address.
			if ip == "" {
				ip = request.RemoteAddr
			}
			return httprate.CanonicalizeIP(ip), nil
		},
		httprate.WithLimitHandler(func(responseWriter http.ResponseWriter, request *http.Request) {
			realIP := middleware.GetClientIP(request.Context())
			fmt.Printf("Blocked request from IP: %s\n", realIP)
			http.Error(responseWriter, "Too many requests.", http.StatusTooManyRequests)
		}),
	)) // 10 request in 5 seconds

	router.Get("/", mainPageHandler)
	router.Get("/{sessionId}", mainPageHandler)
	router.Post("/sse", longSSEHandler)
	router.Post("/new", newChatHandler)
	promptRouter.Post("/chat", promptHandler)
	router.Post("/session/delete", deleteSessionHandler)
	router.Post("/sessions/search", searchSessionHandler)
	router.Post("/fileupload", fileUploadHandler)
	router.Post("/fileupload/remove", removeUploadedFileHandler)
	router.Post("/retry", retryHandler)
	router.Post("/image", getImageHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Mount("/", promptRouter)
	http.ListenAndServe(":3000", router)
}
