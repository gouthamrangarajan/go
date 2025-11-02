package main

import (
	"datastar-web-learnings/components"
	"datastar-web-learnings/models"
	"datastar-web-learnings/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	components.Main().Render(request.Context(), responseWriter)
}

func landingPageDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	offsetStr := chi.URLParam(request, "offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	sse := datastar.NewSSE(responseWriter, request)
	loadVideosWithOffset(offset, true, sse)
}
func emptySearchHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	loadVideosWithOffset(0, false, sse)
}

func searchHandler(responseWriter http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(chi.URLParam(request, "query"))
	// fmt.Printf("Search query received: %v\n", query)
	sse := datastar.NewSSE(responseWriter, request)
	if query == "" {
		loadVideosWithOffset(0, false, sse)
		return
	}
	openAIChannel := make(chan []float32)
	defer close(openAIChannel)
	go services.GetOpenAIEmbeddings(query, openAIChannel)
	vector := <-openAIChannel
	if vector == nil {
		fmt.Printf("No embedding vector received from OpenAI\n")
		return
	}
	pineconeChannel := make(chan []string)
	defer close(pineconeChannel)
	go services.QueryPineconeDb(vector, pineconeChannel)
	videoIds := <-pineconeChannel
	if len(videoIds) == 0 {
		sse.PatchElementTempl(components.PlayerList([]models.VideoResponse{}), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
		sse.RemoveElementByID("loadMore")
		return
	}

	dbChannel := make(chan []models.VideoResponse)
	defer close(dbChannel)
	go services.FilterVideos(request.Context(), videoIds, dbChannel)
	videos := <-dbChannel
	sse.PatchElementTempl(components.PlayerList(videos), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	sse.RemoveElementByID("loadMore")
}

func loadVideosWithOffset(offset int, append bool, sse *datastar.ServerSentEventGenerator) {
	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	channel := make(chan []models.VideoResponse)
	go services.GetVideos(sse.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: offset}, channel)
	videos := <-channel
	if append {
		sse.PatchElementTempl(components.PlayerList(videos), datastar.WithSelector("section"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.PlayerList(videos), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	}

	if len(videos) < noOfItems {
		sse.RemoveElementByID("loadMore")
		return
	}
	if offset == 0 {
		sse.PatchElementTempl(components.LoadMore(noOfItems), datastar.WithSelector("main"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.LoadMore(noOfItems + offset))
	}
}
