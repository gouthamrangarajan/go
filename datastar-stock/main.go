package main

import (
	"fmt"
	"net/http"

	"datastar-stock/components"
	"datastar-stock/services"

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
	router.Use(services.LoggedInMiddleware)
	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		component := components.Landing()
		component.Render(request.Context(), responseWriter)
	})
	router.Post("/login", loginHandler)
	router.Get("/home/populars", popularsDataHandler)
	router.Get("/home/recent", recentDataHandler)
	router.Get("/data/{ticker}", tickerDataHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
