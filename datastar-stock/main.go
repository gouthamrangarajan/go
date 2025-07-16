package main

import (
	"fmt"
	"net/http"

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

	router.Get("/", landingPageHandler)
	router.Post("/login", loginHandler)
	router.Get("/home/populars", popularsPageHandler)
	router.Post("/home/populars/priority/{incrementDecrementIndicator}/{ticker}", popularsPriorityIncrementDecrementHandler)
	router.Get("/home/recent", recentPageHandler)
	router.Get("/data/{ticker}", tickerDataHandler)
	router.Get("/data/multiple/{tickers}", multipleTickerDataHandler)
	router.Post("/home/recent/more", recentDataHandlerWithCount)
	router.Get("/home/recent/add", addRecentUIHandler)
	router.Post("/home/recent/add/{ticker}/{company}", addRecentTickerHandler)
	router.Post("/home/recent/add/close", closeAddRecentHandler)
	router.Post("/companies/search", searchCompaniesHandler)
	router.Get("/companies", companiesPageHandler)
	router.Get("/companies/all/{offset}", companiesAllDataHandler)
	router.Get("/companies/add", addCompanyUIHandler)
	router.Post("/companies/add/close", closeAddCompanyHandler)
	router.Post("/companies/add", addCompanyHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
