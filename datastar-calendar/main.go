package main

import (
	"datastar-calendar/services"
	"datastar-calendar/services/middleware"
	"fmt"
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
	router.Get("/sse", services.SSEHandler)
	router.Get("/{id}", services.DetailsUI)
	router.Get("/detail/close", services.CloseDetailsUI)
	router.Get("/add", services.AddUI)
	router.Get("/add/close", services.CloseAddUI)
	router.Post("/add", services.SaveEvent)
	router.Post("/update", services.SaveEvent)
	router.Get("/wk", services.WeekPage)
	router.Post("/login", services.Login)
	// router.Post("/dnd", services.UpdateDate)
	router.Delete("/delete", services.DeleteEvent)
	router.Get("/assets/*", func(response http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(response, request)
	})

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", router)
}
