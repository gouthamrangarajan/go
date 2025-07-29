package main

import (
	"fmt"
	"net/http"

	"htmx-grocery/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lpernett/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))

	router.Use(services.ChiMiddleware)

	router.Get("/", MainPageWithChi)
	router.Post("/login", Login)
	router.Post("/add", AddGroceryItem)
	router.Post("/delete", RemoveGroceryItem)
	router.Post("/increment", IncrementGroceryItemQuantity)
	router.Post("/decrement", DecrementGroceryItemQuantity)
	router.Post("/complete", ToggleCompleteGroceryItem)
	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", router)
}
