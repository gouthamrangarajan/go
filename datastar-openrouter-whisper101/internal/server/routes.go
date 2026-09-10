package server

import (
	"datastar-openrouter-whisper101/internal/handlers"
	"datastar-openrouter-whisper101/internal/services/openrouter"
	"datastar-openrouter-whisper101/internal/views/pages"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))

	router.Get("/", templ.Handler(pages.Home()).ServeHTTP)

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.ServeFile(responseWriter, request, "assets/"+chi.URLParam(request, "*"))
	})
	openRouterClient := openrouter.NewClient()
	transcribeHandler := handlers.NewTranscribeHandler(openRouterClient)
	router.Post("/transcribe", transcribeHandler.Transcribe)
	return router

}
