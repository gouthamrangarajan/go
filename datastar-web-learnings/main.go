package main

import (
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
	dataRouter := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Use(middleware.Recoverer)

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

	dataRouter.Use(httprate.LimitByIP(rateLimitRequests, time.Duration(rateLimitSeconds)*time.Second)) // 10 request in 5 seconds

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Get("/", landingPageHandler)
	router.Get("/config", configHandler)
	router.Get("/add", addPageHandler)
	router.Post("/add", addVideoHandler)
	router.Post("/tags/ui", tagsUIHandler)
	router.Post("/delete", deleteVideoHandler)

	dataRouter.Get("/data/{offset}", landingPageDataHandler)
	dataRouter.Get("/search/", emptySearchHandler)
	dataRouter.Get("/search/{query}", searchHandler)
	router.Mount("/", dataRouter)
	http.ListenAndServe(":3000", router)
}
