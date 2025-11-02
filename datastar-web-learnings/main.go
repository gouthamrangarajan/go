package main

import (
	"fmt"
	"net/http"
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

	dataRouter.Use(httprate.LimitByIP(3, 5*time.Second)) // 3 request in 5 seconds

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Get("/", landingPageHandler)
	dataRouter.Get("/data/{offset}", landingPageDataHandler)
	dataRouter.Get("/search/", emptySearchHandler)
	dataRouter.Get("/search/{query}", searchHandler)
	router.Mount("/", dataRouter)
	http.ListenAndServe(":3000", router)
}
