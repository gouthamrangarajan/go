package main

import (
	"datastar-portfolio/components"
	"datastar-portfolio/models"
	"datastar-portfolio/services"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/starfederation/datastar-go/datastar"
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
		featureChannel := make(chan []models.DemoItem)
		go services.GetFeaturedDemos(featureChannel)
		features := <-featureChannel
		close(featureChannel)
		component := components.Landing(features)
		component.Render(request.Context(), responseWriter)
	})
	router.Get("/projects", func(responseWriter http.ResponseWriter, request *http.Request) {
		channel := make(chan []models.DemoItem)
		go services.GetAllDemos(channel, true)
		allProjects := <-channel
		close(channel)
		component := components.Projects(allProjects)
		component.Render(request.Context(), responseWriter)
	})
	router.Post("/search", func(responseWriter http.ResponseWriter, request *http.Request) {
		var clientSignal models.ClientSignals
		err := datastar.ReadSignals(request, &clientSignal)
		if err != nil {
			fmt.Printf("Error reading client signals: %v\n", err.Error())
		}
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchSignals([]byte("{filtering:true}"))
		clientSignal.SrchTxt = strings.TrimSpace(clientSignal.SrchTxt)
		if clientSignal.SrchTxt != "" {
			embeddingRequest := models.VoyageEmbeddingRequest{
				Input: []string{clientSignal.SrchTxt},
			}
			embeddingChannel := make(chan models.VoyageEmbeddingResponse)
			go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
			embeddingResponse := <-embeddingChannel
			close(embeddingChannel)
			if len(embeddingResponse.Data) > 0 {
				searchChannel := make(chan []models.DemoItem)
				go services.SearchDemos(searchChannel, embeddingResponse.Data[0].Embedding, clientSignal.ServiceFilter)
				searchResults := <-searchChannel
				close(searchChannel)
				sse.PatchElementTempl(components.ProjectCardCollection(searchResults), datastar.WithUseViewTransitions(true))
				sse.PatchSignals([]byte("{filtering:false}"))
				return
			}
		}
		getAllDataChannel := make(chan []models.DemoItem)
		go services.GetAllDemosWithServiceFilter(getAllDataChannel, clientSignal.ServiceFilter)
		allDemos := <-getAllDataChannel
		close(getAllDataChannel)
		sse.PatchElementTempl(components.ProjectCardCollection(allDemos), datastar.WithUseViewTransitions(true))
		sse.PatchSignals([]byte("{filtering:false}"))
	})

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
}
