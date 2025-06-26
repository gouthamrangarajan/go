package main

import (
	"fmt"
	"htmx-calendar/services"
	"htmx-calendar/services/middleware"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
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
	router.Use(chiMiddleware.Logger)
	router.Use(chiMiddleware.Compress(5))
	router.Use(middleware.Authorization)

	router.Get("/", services.MonthPage)
	router.Get("/add", services.AddPage)
	router.Post("/add", services.AddPage)
	router.Get("/wk", services.WeekPage)
	router.Post("/login", services.Login)
	router.Post("/dnd", services.UpdateDate)
	router.Delete("/delete", services.DeleteEvent)
	router.Get("/assets/*", func(response http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(response, request)
	})

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", router)
}
