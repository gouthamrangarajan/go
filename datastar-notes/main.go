package main

import (
	"fmt"
	"net/http"

	"datastar-notes/services/middlewares"

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
	router.Use(middlewares.Authorization)

	router.Get("/login", loginPageHandler)
	router.Get("/login/retry", loginRetryHandler)
	router.Post("/otp", sendVerificationCodeHandler)
	router.Post("/otp/verify", verifyOTPHandler)

	router.Get("/", getNotesHandler)
	router.Get("/notes/editorui", getNotesHandler)
	router.Post("/notes/save", updateNoteHandler)
	router.Get("/notes/title/edit", getTitleEditUIHandler)
	router.Post("/notes/title/edit", saveTitleHandler)
	router.Post("/notes/add", addNoteHandler)
	router.Post("/notes/delete", deleteNoteHandler)
	router.Get("/notes/reorder", reorderNotesUIHandler)
	router.Post("/notes/reorder", saveReorderedNotesHandler)
	router.Post("/notes/summarize", summarizeNoteHandler)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
