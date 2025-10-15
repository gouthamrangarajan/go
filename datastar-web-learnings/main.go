package main

import (
	"datastar-web-learnings/components"
	"datastar-web-learnings/models"
	"datastar-web-learnings/services"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/starfederation/datastar/sdk/go/datastar"
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
	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		components.Main().Render(request.Context(), responseWriter)
	})
	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	router.Get("/data/{offset}", func(responseWriter http.ResponseWriter, request *http.Request) {
		offsetStr := chi.URLParam(request, "offset")
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			offset = 0
		}
		noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
		noOfItems, err := strconv.Atoi(noOfItemsStr)
		if err != nil {
			noOfItems = 12
		}
		channel := make(chan []models.VideoResponse)
		go services.GetVideos(request.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: offset}, channel)
		videos := <-channel
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.PlayerList(videos), datastar.WithSelector("section"), datastar.WithModeAppend())

		if len(videos) < noOfItems {
			sse.RemoveElementByID("loadMore")
			return
		}
		if offset == 0 {
			sse.PatchElementTempl(components.LoadMore(noOfItems), datastar.WithSelector("main"), datastar.WithModeAppend())
		} else {
			sse.PatchElementTempl(components.LoadMore(noOfItems + offset))
		}
	})
	http.ListenAndServe(":3000", router)
}
