package main

import (
	"context"
	"datastar-web-learnings/components"
	"datastar-web-learnings/models"
	"datastar-web-learnings/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

var sidMap = sync.Map{}
var quizMap = sync.Map{}

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	firebaseConfig := models.FirebaseAuthConfig{
		ApiKey: os.Getenv("FIREBASE_API_KEY"),
		Domain: os.Getenv("FIREBASE_AUTH_DOMAIN"),
	}
	if request.Header.Get("Datastar-Request") == "true" {
		var clientSignal models.UISignals

		datastar.ReadSignals(request, &clientSignal)
		var videos []models.VideoResponse
		if strings.TrimSpace(clientSignal.SearchTxt) == "" {
			videos = services.GetFirstSetOfVideos(request.Context())
		}
		if sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.LANDING_PAGE_UI,
				SearchTxt:        clientSignal.SearchTxt,
				Data:             videos,
			}
		}
		if strings.TrimSpace(clientSignal.SearchTxt) != "" {
			searchVideosAndSendDataToChannel(clientSignal, request.Context())
		}
		return
	}
	components.Landing(firebaseConfig).Render(request.Context(), responseWriter)
}

func sseHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	if clientSignal.Sid == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}

	sessionSseChannel := make(chan models.LongSSEData, 16)
	sidMap.Store(clientSignal.Sid, sessionSseChannel)

	sse := datastar.NewSSE(responseWriter, request)

	var videos []models.VideoResponse
	if strings.TrimSpace(clientSignal.SearchTxt) == "" {
		videos = services.GetFirstSetOfVideos(sse.Context())
		loadVideosWithOffsetUI(models.LongSSEData{Data: videos, OffsetVal: 0}, false, sse)
	} else {
		searchUIForFirstSetData(sse, strings.TrimSpace(clientSignal.SearchTxt))
	}

	heartBeatTicker := time.NewTicker(5 * time.Second)
	defer heartBeatTicker.Stop()

	for {
		select {
		case <-request.Context().Done():
			sidMap.Delete(clientSignal.Sid)
			return
		case sseData := <-sessionSseChannel:
			if channelInMap, ok := sidMap.Load(clientSignal.Sid); !ok || channelInMap != sessionSseChannel {
				return
			}
			switch sseData.FunctionalityVal {
			case models.LANDING_PAGE_UI:
				landingPageUI(sse, sseData)
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
			case models.ADD_PAGE_UI:
				addPageUi(sse)
			case models.TAGS_UI:
				tagsUI(sse, sseData)
			case models.ADD_VIDEO_VALIDATION_ERROR_FUNCTIONALITY:
				addVideoValidationErrorUI(sse, sseData)
			case models.ADD_VIDEO_SUCCESS_FUNCTIONALITY:
				addVideoSuccessUI(sse, sseData)
			case models.ADD_VIDEO_ERROR_FUNCTIONALITY:
				addVideoErrorUI(sse, sseData)
			case models.DELETE_VIDEO_SUCCESS_FUNCTIONALITY:
				deleteVideoSuccessUI(sse, sseData)
			case models.DELETE_VIDEO_ERROR_FUNCTIONALITY:
				deleteVideoErrorUI(sse)
			case models.LOAD_QUIZ_UI_FUNCTIONALITY:
				loadQuizUI(sse, sseData)
			case models.QUIZ_GENERATING_FUNCTIONALITY:
				quizGeneratingUI(sse)
			case models.QUIZ_GENERATION_ERROR_FUNCTIONALTIY:
				quizGenerationErrorUI(sse)
			case models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY:
				quizQuestionUI(sse, sseData)
			}
		case <-heartBeatTicker.C:
			if channelInMap, ok := sidMap.Load(clientSignal.Sid); !ok || channelInMap != sessionSseChannel {
				return
			}
		}
	}
}

func loadMoreHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	channel := make(chan []models.VideoResponse)
	defer close(channel)
	go services.GetVideos(request.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: clientSignal.Offset}, channel)
	videos := <-channel
	if sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid); sidExists {
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
func landingPageUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchElementTempl(components.LandingMainWithoutLoad(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.AddVideoButton(), datastar.WithUseViewTransitions(true))

	loadVideosWithOffsetUI(models.LongSSEData{Data: sseData.Data, OffsetVal: 0}, false, sse)
	if strings.TrimSpace(sseData.SearchTxt) != "" {
		removeLoadMoreUI(sse)
	}
}
func searchHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)
	query := strings.TrimSpace(clientSignal.SearchTxt)
	// fmt.Printf("Search query received: %v\n", query)
	if query == "" {
		videos := services.GetFirstSetOfVideos(request.Context())
		if sessionSseChannel, sidExists := sidMap.Load(clientSignal.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.CLEAR_SEARCH_FUNCTIONALITY,
				Data:             videos,
			}
			return
		}
	}
	searchVideosAndSendDataToChannel(clientSignal, request.Context())

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
func searchVideosAndSendDataToChannel(data models.UISignals, ctxt context.Context) {
	aIResponseChannel := make(chan string)
	defer close(aIResponseChannel)
	go services.VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(data.SearchTxt, aIResponseChannel)
	aIResponse := <-aIResponseChannel
	if aIResponse == "" {
		fmt.Printf("Query not related to technology topics: %v\n", data.SearchTxt)
		if sessionSseChannel, sidExists := sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.INVALID_SEARCH_FUNCTIONALITY,
			}
		}
		return
	}
	vectorChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(vectorChannel)
	go services.CallVoyageEmbedding(models.VoyageEmbeddingRequest{Input: []string{aIResponse}}, vectorChannel)
	vectorResponse := <-vectorChannel
	if len(vectorResponse.Data) == 0 || len(vectorResponse.Data[0].Embedding) == 0 {
		fmt.Printf("No embedding vector received from ai for %v\n", data.SearchTxt)
		if sessionSseChannel, sidExists := sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	pineconeChannel := make(chan []string)
	defer close(pineconeChannel)
	go services.QueryPineconeDb(vectorResponse.Data[0].Embedding, pineconeChannel)
	videoIds := <-pineconeChannel
	if len(videoIds) == 0 {
		if sessionSseChannel, sidExists := sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	dbChannel := make(chan []models.VideoResponse)
	defer close(dbChannel)
	go services.FilterVideos(ctxt, videoIds, dbChannel)
	videos := <-dbChannel
	if len(videos) == 0 {
		if sessionSseChannel, sidExists := sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	if sessionSseChannel, sidExists := sidMap.Load(data.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.SEARCH_FUNCTIONALITY,
			SearchTxt:        data.SearchTxt,
			Data:             videos,
		}
	}
}

func addPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if uiSignals.IdToken != "" {
		channel := make(chan bool)
		defer close(channel)
		go services.VerifyIdToken(request.Context(), uiSignals.IdToken, channel)
		isValidToken := <-channel
		if isValidToken {
			if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
				sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
					FunctionalityVal: models.ADD_PAGE_UI,
				}
			}
			return
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}
func addPageUi(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.AddVideo(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.HomeButton(), datastar.WithUseViewTransitions(true))
}
func tagsHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.TAGS_UI,
			Tags:             uiSignals.Tags,
			UserAgent:        userAgent,
		}
	}
}
func tagsUI(sse *datastar.ServerSentEventGenerator, data models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(data.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.TagsList(data.Tags), datastar.WithUseViewTransitions(useViewTransition))
}

func addVideoHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	uiSignals.Title = strings.TrimSpace(uiSignals.Title)
	uiSignals.Subtitle = strings.TrimSpace(uiSignals.Subtitle)
	uiSignals.VideoId = strings.TrimSpace(uiSignals.VideoId)
	uiSignals.Transcript = strings.TrimSpace(uiSignals.Transcript)

	if uiSignals.IdToken != "" {
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
			// sse := datastar.NewSSE(responseWriter, request)
			if len(errorMessages) > 0 {
				if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal:      models.ADD_VIDEO_VALIDATION_ERROR_FUNCTIONALITY,
						AddVideoErrorMessages: errorMessages,
						AddVideoErrorsignals:  errorSignals,
						UserAgent:             userAgent,
					}
				}
				return
			}

			saveToDbChannel := make(chan bool)
			defer close(saveToDbChannel)
			go services.UpsertVideo(uiSignals, saveToDbChannel)
			success := <-saveToDbChannel
			if success {
				if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal:      models.ADD_VIDEO_SUCCESS_FUNCTIONALITY,
						AddVideoErrorMessages: []string{},
						AddVideoErrorsignals:  "",
						UserAgent:             userAgent,
					}
				}
				deleteDocIdAndVideoIdNotMatchChannel := make(chan bool)
				defer close(deleteDocIdAndVideoIdNotMatchChannel)
				go services.CheckAndDeleteIfDocIdAndVideoIdAreNotSame(uiSignals.VideoId, deleteDocIdAndVideoIdNotMatchChannel)

				dataToVectorize := models.VideoResponse{
					Title:      uiSignals.Title,
					Subtitle:   uiSignals.Subtitle,
					Tags:       trimmedTags,
					Transcript: uiSignals.Transcript,
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
			} else if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
				sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
					FunctionalityVal:      models.ADD_VIDEO_ERROR_FUNCTIONALITY,
					AddVideoErrorMessages: []string{},
					AddVideoErrorsignals:  "",
					UserAgent:             userAgent,
				}
			}
			return
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}

func addVideoValidationErrorUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoValidationError(sseData.AddVideoErrorMessages), datastar.WithUseViewTransitions(useViewTransition))
	sse.PatchSignals([]byte(`{` + sseData.AddVideoErrorsignals + `}`))
}
func addVideoSuccessUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoSuccessResult(), datastar.WithUseViewTransitions(useViewTransition))
	sse.PatchSignals([]byte(`{videoId:'',title:'',subtitle:'',tags:[],rank:1}`))
	sse.PatchElementTempl(components.TagsList([]string{}), datastar.WithUseViewTransitions(useViewTransition))
}
func addVideoErrorUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoErrorResult(), datastar.WithUseViewTransitions(useViewTransition))

}
func deleteVideoHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if uiSignals.IdToken != "" {
		verifyTokenChannel := make(chan bool)
		defer close(verifyTokenChannel)
		go services.VerifyIdToken(request.Context(), uiSignals.IdToken, verifyTokenChannel)
		isValidToken := <-verifyTokenChannel
		if isValidToken {
			if uiSignals.VideoToDelete == "" {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
			deleteVideoChannel := make(chan bool)
			defer close(deleteVideoChannel)
			go services.DeleteVideo(uiSignals.VideoToDelete, deleteVideoChannel)
			success := <-deleteVideoChannel
			if success {
				if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal: models.DELETE_VIDEO_SUCCESS_FUNCTIONALITY,
						SearchTxt:        uiSignals.SearchTxt,
						VideoDeleted:     uiSignals.VideoToDelete,
					}
				}
				deletePineconeRecordChannel := make(chan bool)
				defer close(deletePineconeRecordChannel)
				go services.DeleteRecordPineconeDb(uiSignals.VideoToDelete, deletePineconeRecordChannel)
				<-deletePineconeRecordChannel
			} else if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
				sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
					FunctionalityVal: models.DELETE_VIDEO_ERROR_FUNCTIONALITY,
					SearchTxt:        uiSignals.SearchTxt,
					VideoDeleted:     uiSignals.VideoToDelete,
				}
			}
			return
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}

func deleteVideoSuccessUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchElementTempl(components.EmptyDeleteVideoResult(), datastar.WithUseViewTransitions(true))
	sse.PatchSignals([]byte(`{videoToDelete:'',showDeleteConfirm:false}`))
	time.Sleep(200 * time.Millisecond) //wait for UI animation
	sse.RemoveElement("#playerContainer_"+sseData.VideoDeleted+"_"+sseData.SearchTxt, datastar.WithUseViewTransitions(true))
}

func deleteVideoErrorUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.DeleteVideoErrorResult(), datastar.WithUseViewTransitions(false))
}
func loadQuizHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if strings.TrimSpace(uiSignals.QuizVideoId) == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.LOAD_QUIZ_UI_FUNCTIONALITY,
			QuizVideoId:      uiSignals.QuizVideoId,
		}
	}
}
func loadQuizUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchSignals([]byte(`{_loadingQuiz:false,_showQuiz:true}`))
	dbChannel := make(chan []models.VideoResponse)
	defer close(dbChannel)
	go services.FilterVideos(sse.Context(), []string{sseData.QuizVideoId}, dbChannel)
	quizVideos := <-dbChannel
	transcript := ""
	if len(quizVideos) > 0 && strings.TrimSpace(quizVideos[0].Transcript) != "" {
		transcript = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(quizVideos[0].Transcript, "'", "\\'"), "\n", "\\n"))
	}
	sse.PatchElementTempl(components.CreateQuizForm(transcript),
		datastar.WithModeInner(), datastar.WithSelector("#quizDialog"))
}
func quizGeneratingUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.QuizGenerating(), datastar.WithSelector("#quizDialog"), datastar.WithModeInner())
}
func quizGenerationAndPrevNextHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	transcript := strings.TrimSpace(uiSignals.Transcript)
	if transcript == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	// fmt.Printf("Transcript %v\n",transcript)
	if quizMapItem, quizMapExists := quizMap.Load(uiSignals.Sid); quizMapExists {
		quizResponse := quizMapItem.(models.QuizResponse)
		if quizResponse.VideoId == uiSignals.QuizVideoId {
			if uiSignals.QuizIndex < 0 || uiSignals.QuizIndex >= len(quizResponse.Questions) {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
			if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
				sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
					FunctionalityVal: models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY,
					QuizIndex:        uiSignals.QuizIndex,
					Sid:              uiSignals.Sid,
				}
			}
			return
		}
	}
	if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.QUIZ_GENERATING_FUNCTIONALITY,
		}
	}
	openRouterChannel := make(chan models.QuizResponse)
	defer close(openRouterChannel)
	go services.GenerateQuizUsingOpenRouter(uiSignals, openRouterChannel)

	updateTranscriptChannel := make(chan bool)
	defer close(updateTranscriptChannel)
	go services.UpdateTranscriptForQuiz(uiSignals, updateTranscriptChannel)

	quizResponse := <-openRouterChannel
	if len(quizResponse.Questions) > 0 {
		quizResponse.VideoId = uiSignals.QuizVideoId
		quizMap.Store(uiSignals.Sid, quizResponse)

		if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY,
				QuizIndex:        0,
				Sid:              uiSignals.Sid,
			}
		}
	} else {
		if sessionSseChannel, sidExists := sidMap.Load(uiSignals.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.QUIZ_GENERATION_ERROR_FUNCTIONALTIY,
			}
		}
	}
	<-updateTranscriptChannel
}

func quizQuestionUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	quizResponse, _ := quizMap.Load(sseData.Sid)
	sse.PatchElementTempl(components.QuizQuestion(quizResponse.(models.QuizResponse), sseData.QuizIndex),
		datastar.WithSelector("#quizDialog"),
		datastar.WithModeInner())
}

func quizGenerationErrorUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.QuizGenerationError(),
		datastar.WithSelector("#quizDialog"),
		datastar.WithModeInner())
}
