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
	openAIResponseChannel := make(chan bool)
	defer close(openAIResponseChannel)
	go services.VerifyTechnologyTopicsSearch(query, openAIResponseChannel)
	openAIResponse := <-openAIResponseChannel
	if !openAIResponse {
		fmt.Printf("Query not related to technology topics: %v\n", query)
		sse.PatchElementTempl(components.NoDataFound("Looks like your search isn’t technology-related. Please try a tech-related query."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
		removeLoaderMore(sse)
		return
	}
	openAIVectorChannel := make(chan []float32)
	defer close(openAIVectorChannel)
	go services.GetOpenAIEmbeddings(query, openAIVectorChannel)
	vector := <-openAIVectorChannel
	if vector == nil {
		fmt.Printf("No embedding vector received from OpenAI for %v\n", query)
		sse.PatchElementTempl(components.NoDataFound("No technology videos found matching your search."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
		removeLoaderMore(sse)
		return
	}
	pineconeChannel := make(chan []string)
	defer close(pineconeChannel)
	go services.QueryPineconeDb(vector, pineconeChannel)
	videoIds := <-pineconeChannel
	if len(videoIds) == 0 {
		sse.PatchElementTempl(components.NoDataFound("No technology videos found matching your search."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
		removeLoaderMore(sse)
		return
	}

	dbChannel := make(chan []models.VideoResponse)
	defer close(dbChannel)
	go services.FilterVideos(request.Context(), videoIds, dbChannel)
	videos := <-dbChannel
	if len(videos) == 0 {
		sse.PatchElementTempl(components.NoDataFound("No technology videos found matching your search."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	} else {
		sse.PatchElementTempl(components.PlayerList(videos), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	}
	removeLoaderMore(sse)
}
func removeLoaderMore(sse *datastar.ServerSentEventGenerator) {
	sse.ExecuteScript("document.getElementById('loadMore')?.remove();", datastar.WithExecuteScriptAutoRemove(true))
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
		// sse.RemoveElementByID("loadMore")
		sse.ExecuteScript("document.getElementById('loadMore')?.remove();", datastar.WithExecuteScriptAutoRemove(true))
		return
	}
	if offset == 0 {
		sse.PatchElementTempl(components.LoadMore(noOfItems), datastar.WithSelector("main"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.LoadMore(noOfItems + offset))
	}
}
