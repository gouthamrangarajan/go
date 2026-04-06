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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

var sidMap = sync.Map{}

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	firebaseConfig := models.FirebaseAuthConfig{
		ApiKey: os.Getenv("FIREBASE_API_KEY"),
		Domain: os.Getenv("FIREBASE_AUTH_DOMAIN"),
	}
	if request.Header.Get("Datastar-Request") == "true" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(components.LandingMain(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.AddVideoButton(), datastar.WithUseViewTransitions(true))
		return
	}
	newSid := uuid.New().String()
	sidMap.Store(newSid, make(chan models.LongSSEData))
	components.Landing(firebaseConfig, newSid).Render(request.Context(), responseWriter)
}
func sseHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	if clientSignal.Sid == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid)
	if !sidExists {
		sessionSseChannel = make(chan models.LongSSEData)
		sidMap.Store(clientSignal.Sid, sessionSseChannel)
	}
	sse := datastar.NewSSE(responseWriter, request)

	var videos []models.VideoResponse
	if strings.TrimSpace(clientSignal.SearchTxt) == "" {
		videos = services.GetFirstSetOfVideos(sse.Context())
		loadVideosWithOffsetUI(models.LongSSEData{Data: videos, OffsetVal: 0}, false, sse)
	} else {
		searchUIForFirstSetData(sse, strings.TrimSpace(clientSignal.SearchTxt))
	}

	for {
		select {
		case <-request.Context().Done():
			close(sessionSseChannel.(chan models.LongSSEData))
			sidMap.Delete(clientSignal.Sid)
			return
		case sseData := <-sessionSseChannel.(chan models.LongSSEData):
			switch sseData.FunctionalityVal {
			case models.LOAD_MORE_FUNCTIONALITY:
				loadVideosWithOffsetUI(sseData, true, sse)
			case models.SEARCH_FUNCTIONALITY:
				loadVideosWithOffsetUI(sseData, false, sse)
				removeLoadMoreUI(sse)
			case models.CLEAR_SEARCH_FUNCTIONALITY:
				loadVideosWithOffsetUI(sseData, false, sse)
			case models.INVALID_SEARCH_FUNCTIONALITY:
				invalidSearchUI(sse)
				removeLoadMoreUI(sse)
			case models.NO_DATA_FOUND_FUNCTIONALITY:
				noDataFoundUI(sse)
				removeLoadMoreUI(sse)
			}
		}
	}
}

func loadMoreHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	if sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid); sidExists {
		noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
		noOfItems, err := strconv.Atoi(noOfItemsStr)
		if err != nil {
			noOfItems = 12
		}
		channel := make(chan []models.VideoResponse)
		defer close(channel)
		go services.GetVideos(request.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: clientSignal.Offset}, channel)
		videos := <-channel
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.LOAD_MORE_FUNCTIONALITY,
			OffsetVal:        clientSignal.Offset,
			Data:             videos,
		}
	}

}
func loadVideosWithOffsetUI(sseData models.LongSSEData, append bool, sse *datastar.ServerSentEventGenerator) {
	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}

	if append {
		sse.PatchElementTempl(components.PlayerList(sseData.Data, sseData.SearchTxt), datastar.WithSelector("section"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.PlayerList(sseData.Data, sseData.SearchTxt), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	}

	if len(sseData.Data) < noOfItems {
		removeLoadMoreUI(sse)
		return
	}
	addAppendLoadMoreUI(sse, sseData.OffsetVal)
}
func addAppendLoadMoreUI(sse *datastar.ServerSentEventGenerator, offset int) {
	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	if offset == 0 {
		removeLoadMoreUI(sse)
		sse.PatchElementTempl(components.LoadMore(noOfItems), datastar.WithSelector("main"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.LoadMore(noOfItems + offset))
	}
}
func searchUIForFirstSetData(sse *datastar.ServerSentEventGenerator, query string) {
	aIResponseChannel := make(chan string)
	go services.VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(query, aIResponseChannel)
	aIResponse := <-aIResponseChannel
	close(aIResponseChannel)
	if aIResponse == "" {
		fmt.Printf("Query not related to technology topics: %v\n", query)
		invalidSearchUI(sse)
		return
	}
	vectorChannel := make(chan models.VoyageEmbeddingResponse)
	go services.CallVoyageEmbedding(models.VoyageEmbeddingRequest{Input: []string{aIResponse}}, vectorChannel)
	vectorResponse := <-vectorChannel
	close(vectorChannel)
	if len(vectorResponse.Data) == 0 || len(vectorResponse.Data[0].Embedding) == 0 {
		fmt.Printf("No embedding vector received from ai for %v\n", query)
		noDataFoundUI(sse)
		return
	}
	pineconeChannel := make(chan []string)
	go services.QueryPineconeDb(vectorResponse.Data[0].Embedding, pineconeChannel)
	videoIds := <-pineconeChannel
	close(pineconeChannel)
	if len(videoIds) == 0 {
		noDataFoundUI(sse)
		return
	}
	dbChannel := make(chan []models.VideoResponse)
	go services.FilterVideos(sse.Context(), videoIds, dbChannel)
	videos := <-dbChannel
	close(dbChannel)
	if len(videos) == 0 {
		noDataFoundUI(sse)
		return
	}
	loadVideosWithOffsetUI(models.LongSSEData{Data: videos, OffsetVal: 0, SearchTxt: query}, false, sse)
	removeLoadMoreUI(sse)
}
func searchHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)
	query := strings.TrimSpace(clientSignal.SearchTxt)
	// fmt.Printf("Search query received: %v\n", query)
	if sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid); sidExists {
		if query == "" {
			videos := services.GetFirstSetOfVideos(request.Context())
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.CLEAR_SEARCH_FUNCTIONALITY,
				Data:             videos,
			}
			return
		}
		aIResponseChannel := make(chan string)
		defer close(aIResponseChannel)
		go services.VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(query, aIResponseChannel)
		aIResponse := <-aIResponseChannel
		if aIResponse == "" {
			fmt.Printf("Query not related to technology topics: %v\n", query)
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.INVALID_SEARCH_FUNCTIONALITY,
			}
			return
		}
		vectorChannel := make(chan models.VoyageEmbeddingResponse)
		defer close(vectorChannel)
		go services.CallVoyageEmbedding(models.VoyageEmbeddingRequest{Input: []string{aIResponse}}, vectorChannel)
		vectorResponse := <-vectorChannel
		if len(vectorResponse.Data) == 0 || len(vectorResponse.Data[0].Embedding) == 0 {
			fmt.Printf("No embedding vector received from ai for %v\n", query)
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
			return
		}
		pineconeChannel := make(chan []string)
		defer close(pineconeChannel)
		go services.QueryPineconeDb(vectorResponse.Data[0].Embedding, pineconeChannel)
		videoIds := <-pineconeChannel
		if len(videoIds) == 0 {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
			return
		}
		dbChannel := make(chan []models.VideoResponse)
		defer close(dbChannel)
		go services.FilterVideos(request.Context(), videoIds, dbChannel)
		videos := <-dbChannel
		if len(videos) == 0 {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
			return
		}
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.SEARCH_FUNCTIONALITY,
			SearchTxt:        query,
			Data:             videos,
		}
	}

}

func noDataFoundUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.NoDataFound("No technology videos found matching your search."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
}
func invalidSearchUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.NoDataFound("Looks like your search isn’t technology-related. Please try a tech-related query."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
}

func removeLoadMoreUI(sse *datastar.ServerSentEventGenerator) {
	sse.ExecuteScript("document.getElementById('loadMore')?.remove();", datastar.WithExecuteScriptAutoRemove(true))
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

					dataToVectorize := models.VideoResponse{
						Title:    uiSignals.Title,
						Subtitle: uiSignals.Subtitle,
						Tags:     trimmedTags,
					}
					dataToVectorize = services.ConstructTextToVectorize(dataToVectorize, ytResponse.Items[0].Snippet.Description)
					vectorChannel := make(chan models.VoyageEmbeddingResponse)
					defer close(vectorChannel)
					go services.CallVoyageEmbedding(models.VoyageEmbeddingRequest{Input: []string{dataToVectorize.TextToVectorize}}, vectorChannel)
					vectorData := <-vectorChannel
					if len(vectorData.Data) != 0 && len(vectorData.Data[0].Embedding) != 0 {
						upsertPineconeChannel := make(chan int)
						defer close(upsertPineconeChannel)
						go services.UpsertPineconeDb(uiSignals.VideoId, vectorData.Data[0].Embedding, upsertPineconeChannel)
						<-upsertPineconeChannel
						// fmt.Printf("Text vectorized and upserted to Pinecone: %v\n", dataToVectorize.TextToVectorize)
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
					sse.RemoveElement("#playerContainer_"+uiSignals.VideoToDelete+"_"+uiSignals.SearchTxt, datastar.WithUseViewTransitions(true))
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
