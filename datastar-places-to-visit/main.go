package main

import (
	"datastar-placestovisit/components"
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
		fmt.Println("Loaded .env file")
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		components.Main().Render(request.Context(), responseWriter)
	})
	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Get("/map/initialize", initializeMap)
	router.Get("/search", searchCityStateCountry)
	router.Get("/{city}/{lat}/{lng}", getPlaces)
	http.ListenAndServe(":3000", router)
}
