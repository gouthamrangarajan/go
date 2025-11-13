package main

import (
	"datastar-web-learnings/components"
	"datastar-web-learnings/models"
	"datastar-web-learnings/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Datastar-Request") == "true" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.LandingMain(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.AddVideoButton(), datastar.WithUseViewTransitions(true))
		return
	}
	authConfig := models.FirebaseAuthConfig{
		ApiKey: os.Getenv("FIREBASE_API_KEY"),
		Domain: os.Getenv("FIREBASE_AUTH_DOMAIN"),
	}
	components.Landing(authConfig).Render(request.Context(), responseWriter)
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
	openAIResponseChannel := make(chan string)
	defer close(openAIResponseChannel)
	go services.VerifyTechnologyTopicsSearchAndOptimizeQuery(query, openAIResponseChannel)
	openAIResponse := <-openAIResponseChannel
	if openAIResponse == "" {
		fmt.Printf("Query not related to technology topics: %v\n", query)
		sse.PatchElementTempl(components.NoDataFound("Looks like your search isn’t technology-related. Please try a tech-related query."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
		removeLoaderMore(sse)
		return
	}
	openAIVectorChannel := make(chan []float32)
	defer close(openAIVectorChannel)
	go services.GetOpenAIEmbeddings(openAIResponse, openAIVectorChannel)
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
		sse.PatchElementTempl(components.PlayerList(videos, query), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
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
	defer close(channel)
	go services.GetVideos(sse.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: offset}, channel)
	videos := <-channel
	if append {
		sse.PatchElementTempl(components.PlayerList(videos, ""), datastar.WithSelector("section"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.PlayerList(videos, ""), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
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
func addPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	uiSignalsString := request.URL.Query().Get("datastar")
	var uiSignals models.UISignals
	err := json.Unmarshal([]byte(uiSignalsString), &uiSignals)
	if err == nil && uiSignals.IdToken != "" {
		channel := make(chan bool)
		defer close(channel)
		go services.VerifyIdToken(request.Context(), uiSignals.IdToken, channel)
		isValidToken := <-channel
		if isValidToken {
			sse := datastar.NewSSE(responseWriter, request)
			sse.PatchElementTempl(components.AddVideo(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
			sse.PatchElementTempl(components.HomeButton(), datastar.WithUseViewTransitions(true))
			return
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}
func tagsUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	useViewTransition := true
	if strings.Contains(strings.ToLower(userAgent), "mobile") {
		useViewTransition = false
	}
	uiSignalsBytes, _ := io.ReadAll(request.Body)
	uiSignals := models.UISignals{}
	_ = json.Unmarshal(uiSignalsBytes, &uiSignals)
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.TagsList(uiSignals.Tags), datastar.WithUseViewTransitions(useViewTransition))
}

func addVideoHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	useViewTransition := true
	if strings.Contains(strings.ToLower(userAgent), "mobile") {
		useViewTransition = false
	}
	uiSignalsBytes, err := io.ReadAll(request.Body)
	if err == nil {
		var uiSignals models.UISignals
		err = json.Unmarshal(uiSignalsBytes, &uiSignals)
		if err == nil && uiSignals.IdToken != "" {
			verifyTokenChannel := make(chan bool)
			defer close(verifyTokenChannel)
			go services.VerifyIdToken(request.Context(), uiSignals.IdToken, verifyTokenChannel)
			isValidToken := <-verifyTokenChannel
			if isValidToken {
				// fmt.Printf("received add video request: %v\n", uiSignals)
				errorMessages := []string{}
				errorSignals := ""
				if len(strings.TrimSpace(uiSignals.Title)) < 3 {
					errorMessages = append(errorMessages, "Please enter a title of at least 3 characters.")
					errorSignals += "titleError:true,"
				}
				trimmedTags := []string{}
				for _, tag := range uiSignals.Tags {
					trimmedTag := strings.TrimSpace(tag)
					if trimmedTag != "" {
						trimmedTags = append(trimmedTags, trimmedTag)
					}
				}
				if len(trimmedTags) == 0 {
					errorMessages = append(errorMessages, "Please add at least one meaningful tag to describe your video.")
					errorSignals += "tagsError:true,"
				}
				if uiSignals.Rank < 1 || uiSignals.Rank > 5 {
					errorMessages = append(errorMessages, "Please enter a valid rank between 1 and 5.")
					errorSignals += "rankError:true,"
				}
				if len(strings.TrimSpace(uiSignals.VideoId)) != 11 {
					errorMessages = append(errorMessages, "Please enter a valid YouTube video ID.")
					errorSignals += "videoIdError:true,"
				}
				var ytResponse models.YoutubeVideoSearchResponse
				if len(errorMessages) == 0 {
					ytVideoSearchChannel := make(chan models.YoutubeVideoSearchResponse)
					defer close(ytVideoSearchChannel)
					go services.GetYTVideoResponse(uiSignals.VideoId, ytVideoSearchChannel)
					ytResponse = <-ytVideoSearchChannel
					if len(ytResponse.Items) == 0 || ytResponse.Items[0].Id == "" {
						errorMessages = append(errorMessages, "Please enter a valid YouTube video ID.")
						errorSignals += "videoIdError:true,"
					}
				}
				sse := datastar.NewSSE(responseWriter, request)
				if len(errorMessages) > 0 {
					sse.PatchElementTempl(components.AddVideoValidationError(errorMessages), datastar.WithUseViewTransitions(useViewTransition))
					sse.PatchSignals([]byte(`{` + errorSignals + `}`))
					return
				}

				saveToDbChannel := make(chan bool)
				defer close(saveToDbChannel)
				go services.UpsertVideo(uiSignals, saveToDbChannel)
				success := <-saveToDbChannel
				if success {
					sse.PatchElementTempl(components.AddVideoSuccessResult(), datastar.WithUseViewTransitions(useViewTransition))
					sse.PatchSignals([]byte(`{videoId:'',title:'',subtitle:'',tags:[],rank:1}`))
					sse.PatchElementTempl(components.TagsList([]string{}), datastar.WithUseViewTransitions(useViewTransition))

					deleteDocIdAndVideoIdNotMatchChannel := make(chan bool)
					defer close(deleteDocIdAndVideoIdNotMatchChannel)
					go services.CheckAndDeleteIfDocIdAndVideoIdAreNotSame(uiSignals.VideoId, deleteDocIdAndVideoIdNotMatchChannel)

					textToVectorize := uiSignals.Title + " " + uiSignals.Subtitle + " " + strings.Join(trimmedTags, " ") + " " + ytResponse.Items[0].Snippet.Description
					openAIVectorChannel := make(chan []float32)
					defer close(openAIVectorChannel)
					go services.GetOpenAIEmbeddings(textToVectorize, openAIVectorChannel)
					vector := <-openAIVectorChannel
					if vector != nil {
						upsertPineconeChannel := make(chan int)
						defer close(upsertPineconeChannel)
						go services.UpsertPineconeDb(uiSignals.VideoId, vector, upsertPineconeChannel)
						<-upsertPineconeChannel
						// fmt.Printf("Text vectorized and upserted to Pinecone: %v\n", textToVectorize)
					}
					<-deleteDocIdAndVideoIdNotMatchChannel
				} else {
					sse.PatchElementTempl(components.AddVideoErrorResult(), datastar.WithUseViewTransitions(useViewTransition))
				}
				return
			}
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}

func deleteVideoHandler(responseWriter http.ResponseWriter, request *http.Request) {
	uiSignalsBytes, err := io.ReadAll(request.Body)
	if err == nil {
		var uiSignals models.UISignals
		err = json.Unmarshal(uiSignalsBytes, &uiSignals)
		if err == nil && uiSignals.IdToken != "" {
			verifyTokenChannel := make(chan bool)
			defer close(verifyTokenChannel)
			go services.VerifyIdToken(request.Context(), uiSignals.IdToken, verifyTokenChannel)
			isValidToken := <-verifyTokenChannel
			if isValidToken {
				if uiSignals.VideoToDelete == "" {
					http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
					return
				}
				sse := datastar.NewSSE(responseWriter, request)
				deleteVideoChannel := make(chan bool)
				defer close(deleteVideoChannel)
				go services.DeleteVideo(uiSignals.VideoToDelete, deleteVideoChannel)
				success := <-deleteVideoChannel
				if success {
					sse.PatchElementTempl(components.EmptyDeleteVideoResult(), datastar.WithUseViewTransitions(true))
					sse.PatchSignals([]byte(`{videoToDelete:'',showDeleteConfirm:false}`))
					time.Sleep(200 * time.Millisecond) //wait for UI animation
					sse.RemoveElement("#playerContainer_"+uiSignals.VideoToDelete, datastar.WithUseViewTransitions(true))
					deletePineconeRecordChannel := make(chan bool)
					defer close(deletePineconeRecordChannel)
					go services.DeleteRecordPineconeDb(uiSignals.VideoToDelete, deletePineconeRecordChannel)
					<-deletePineconeRecordChannel
				} else {
					sse.PatchElementTempl(components.DeleteVideoErrorResult(), datastar.WithUseViewTransitions(false))
				}
			}

		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}
