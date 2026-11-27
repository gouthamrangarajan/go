package server

import (
	"datastar-calendar/internal/handlers"
	"datastar-calendar/services/db"
	"datastar-calendar/services/middleware"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type Router struct {
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) HttpHandler() http.Handler {
	router := chi.NewRouter()
	router.Use(chiMiddleware.Logger)
	router.Use(chiMiddleware.Compress(5))
	router.Use(middleware.Authorization)

	var uidSidMap sync.Map
	dbService := db.NewDBSerice()
	loginHandler := handlers.NewLoginHandler(dbService)
	monthHandler := handlers.NewMonthHandler(dbService, &uidSidMap)
	weekHandler := handlers.NewWeekHandler(dbService, &uidSidMap)
	actionHandler := handlers.NewActionHandler(dbService, &uidSidMap)
	sseHandler := handlers.NewSSEHandler(dbService, &uidSidMap)
	addUIHandler := handlers.NewAddUIHandler(dbService, &uidSidMap)
	editUIHandler := handlers.NewEditUIHandler(dbService, &uidSidMap)
	manageEventsHandler := handlers.NewManageEventsHandler(dbService, &uidSidMap)

	router.Get("/", monthHandler.Handle)
	router.Get("/sse", sseHandler.Handle)
	router.Get("/add", addUIHandler.Handle)
	router.Get("/add/close", addUIHandler.Close)
	router.Post("/add", actionHandler.Save)
	router.Get("/{id}", editUIHandler.Handle)
	router.Get("/detail/close", editUIHandler.Close)
	router.Post("/update", actionHandler.Save)
	router.Get("/wk", weekHandler.Handle)
	router.Post("/login", loginHandler.Handle)
	router.Post("/delete", actionHandler.Delete)
	router.Get("/manage", manageEventsHandler.Handle)
	router.Post("/manage/filter", manageEventsHandler.Filter)
	router.Post("/events/{id}/status", actionHandler.UpdateStatus)
	router.Get("/assets/*", func(response http.ResponseWriter, request *http.Request) {
		fileServer := http.StripPrefix("/assets/", http.FileServer(http.Dir("assets")))
		fileServer.ServeHTTP(response, request)
	})
	return router
}
