package handlers

import (
	"context"
	"datastar-web-learnings/internal/models"
	"datastar-web-learnings/internal/views/components"
	"datastar-web-learnings/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	openRouter "datastar-web-learnings/services/open-router"
	"datastar-web-learnings/services/pinecone"
	"datastar-web-learnings/services/storage"
	voyage "datastar-web-learnings/services/voyage"

	"github.com/starfederation/datastar-go/datastar"
)

type MainHandler struct {
	sidMap           *sync.Map
	quizMap          *sync.Map
	apiKey           string
	domain           string
	helperService    *services.HelperService
	noOfDbItems      int
	openRouterClient *openRouter.Client
	voyageClient     *voyage.Client
	pineconeClient   *pinecone.Client
	dbService        *storage.DbService
	yTService        *services.YTService
}

func NewMainHandler(sidMap *sync.Map, quizMap *sync.Map, helperService *services.HelperService,
	openRouterclient *openRouter.Client, voyageClient *voyage.Client, pineconeClient *pinecone.Client,
	dbService *storage.DbService, ytService *services.YTService) *MainHandler {

	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	return &MainHandler{
		sidMap:           sidMap,
		quizMap:          quizMap,
		apiKey:           os.Getenv("FIREBASE_API_KEY"),
		domain:           os.Getenv("FIREBASE_AUTH_DOMAIN"),
		helperService:    helperService,
		noOfDbItems:      noOfItems,
		openRouterClient: openRouterclient,
		voyageClient:     voyageClient,
		pineconeClient:   pineconeClient,
		dbService:        dbService,
		yTService:        ytService,
	}
}

func (m *MainHandler) HandleLandingPage(responseWriter http.ResponseWriter, request *http.Request) {
	firebaseConfig := models.FirebaseAuthConfig{
		ApiKey: m.apiKey,
		Domain: m.domain,
	}
	if request.Header.Get("Datastar-Request") == "true" {
		var clientSignal models.UISignals

		datastar.ReadSignals(request, &clientSignal)
		var videos []models.VideoResponse
		if strings.TrimSpace(clientSignal.SearchTxt) == "" {
			videos = m.helperService.GetFirstSetOfVideos(request.Context())
		}
		if sessionSseChannel, sidExists := m.sidMap.Load(clientSignal.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.LANDING_PAGE_UI,
				SearchTxt:        clientSignal.SearchTxt,
				Data:             videos,
			}
		}
		if strings.TrimSpace(clientSignal.SearchTxt) != "" {
			m.searchVideosAndSendDataToChannel(clientSignal, request.Context())
		}
		return
	}
	components.Landing(firebaseConfig).Render(request.Context(), responseWriter)
}

func (m *MainHandler) HandleSSE(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	if clientSignal.Sid == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}

	sessionSseChannel := make(chan models.LongSSEData, 16)
	m.sidMap.Store(clientSignal.Sid, sessionSseChannel)
	defer m.sidMap.CompareAndDelete(clientSignal.Sid, sessionSseChannel)

	responseWriter.Header().Set("Content-Type", "text/event-stream")
	responseWriter.Header().Set("Cache-Control", "no-cache")
	responseWriter.Header().Set("Connection", "keep-alive")
	// This header tells Nginx/Railway Proxy not to buffer the stream
	responseWriter.Header().Set("X-Accel-Buffering", "no")

	sse := datastar.NewSSE(responseWriter, request)

	var videos []models.VideoResponse
	if strings.TrimSpace(clientSignal.SearchTxt) == "" {
		videos = m.helperService.GetFirstSetOfVideos(sse.Context())
		m.loadVideosWithOffsetUI(models.LongSSEData{Data: videos, OffsetVal: 0}, false, sse)
	} else {
		m.searchUIForFirstSetData(sse, strings.TrimSpace(clientSignal.SearchTxt))
	}

	heartBeatTicker := time.NewTicker(5 * time.Second)
	defer heartBeatTicker.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case sseData := <-sessionSseChannel:
			if channelInMap, ok := m.sidMap.Load(clientSignal.Sid); !ok || channelInMap != sessionSseChannel {
				return
			}
			switch sseData.FunctionalityVal {
			case models.LANDING_PAGE_UI:
				m.landingPageUI(sse, sseData)
			case models.LOAD_MORE_FUNCTIONALITY:
				m.loadVideosWithOffsetUI(sseData, true, sse)
			case models.SEARCH_FUNCTIONALITY:
				m.loadVideosWithOffsetUI(sseData, false, sse)
				m.removeLoadMoreUI(sse)
			case models.CLEAR_SEARCH_FUNCTIONALITY:
				m.loadVideosWithOffsetUI(sseData, false, sse)
			case models.INVALID_SEARCH_FUNCTIONALITY:
				m.invalidSearchUI(sse)
				m.removeLoadMoreUI(sse)
			case models.NO_DATA_FOUND_FUNCTIONALITY:
				m.noDataFoundUI(sse)
				m.removeLoadMoreUI(sse)
			case models.ADD_PAGE_UI:
				m.addPageUi(sse)
			case models.TAGS_UI:
				m.tagsUI(sse, sseData)
			case models.ADD_VIDEO_VALIDATION_ERROR_FUNCTIONALITY:
				m.addVideoValidationErrorUI(sse, sseData)
			case models.ADD_VIDEO_SUCCESS_FUNCTIONALITY:
				m.addVideoSuccessUI(sse, sseData)
			case models.ADD_VIDEO_ERROR_FUNCTIONALITY:
				m.addVideoErrorUI(sse, sseData)
			case models.DELETE_VIDEO_SUCCESS_FUNCTIONALITY:
				m.deleteVideoSuccessUI(sse, sseData)
			case models.DELETE_VIDEO_ERROR_FUNCTIONALITY:
				m.deleteVideoErrorUI(sse)
			case models.LOAD_QUIZ_UI_FUNCTIONALITY:
				m.loadQuizUI(sse, sseData)
			case models.QUIZ_GENERATING_FUNCTIONALITY:
				m.quizGeneratingUI(sse)
			case models.QUIZ_GENERATION_ERROR_FUNCTIONALTIY:
				m.quizGenerationErrorUI(sse)
			case models.QUIZ_VERIFY_ANSWER_FUNCTIONALITY:
				m.quizAnswerVerificationUI(sse, sseData)
			case models.QUIZ_VERIFY_ANSWER_ERROR:
				m.quizAnswerVerificationErrorUI(sse, sseData)
			case models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY:
				m.quizQuestionUI(sse, sseData)
			}
		case <-heartBeatTicker.C:
			if channelInMap, ok := m.sidMap.Load(clientSignal.Sid); !ok || channelInMap != sessionSseChannel {
				return
			}
			sse.Send(datastar.EventType("heartbeat"), []string{fmt.Sprintf(": heartbeat %d\n\n", time.Now().Unix())})
		}
	}
}

func (m *MainHandler) HandleLoadMore(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)

	noOfItemsStr := os.Getenv("ITEMS_PER_PAGE")
	noOfItems, err := strconv.Atoi(noOfItemsStr)
	if err != nil {
		noOfItems = 12
	}
	channel := make(chan []models.VideoResponse)
	go m.dbService.GetVideos(request.Context(), models.GetVideosRequest{Limit: noOfItems, Offset: clientSignal.Offset}, channel)
	videos := <-channel
	if sessionSseChannel, sidExists := m.sidMap.Load(clientSignal.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.LOAD_MORE_FUNCTIONALITY,
			OffsetVal:        clientSignal.Offset,
			Data:             videos,
		}
	}

}
func (m *MainHandler) loadVideosWithOffsetUI(sseData models.LongSSEData, append bool, sse *datastar.ServerSentEventGenerator) {

	if append {
		sse.PatchElementTempl(components.PlayerList(sseData.Data, sseData.SearchTxt), datastar.WithSelector("section"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.PlayerList(sseData.Data, sseData.SearchTxt), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
	}

	if len(sseData.Data) < m.noOfDbItems {
		m.removeLoadMoreUI(sse)
		return
	}
	m.addAppendLoadMoreUI(sse, sseData.OffsetVal)
}
func (m *MainHandler) addAppendLoadMoreUI(sse *datastar.ServerSentEventGenerator, offset int) {
	if offset == 0 {
		m.removeLoadMoreUI(sse)
		sse.PatchElementTempl(components.LoadMore(m.noOfDbItems), datastar.WithSelector("main"), datastar.WithModeAppend())
	} else {
		sse.PatchElementTempl(components.LoadMore(m.noOfDbItems + offset))
	}
}
func (m *MainHandler) searchUIForFirstSetData(sse *datastar.ServerSentEventGenerator, query string) {
	decisionChannel := make(chan bool)
	go m.openRouterClient.VerifyTechnologyTopicSearch(query, decisionChannel)
	isTechnology := <-decisionChannel
	if !isTechnology {
		fmt.Printf("Query not related to technology topics: %v\n", query)
		m.invalidSearchUI(sse)
		return
	}
	aIResponseChannel := make(chan string)
	go m.openRouterClient.OptimizeQueryForSearch(query, aIResponseChannel)
	aIResponse := <-aIResponseChannel
	if strings.TrimSpace(aIResponse) == "" {
		aIResponse = query
	}
	vectorChannel := make(chan voyage.Response)
	go m.voyageClient.CallEmbedding(voyage.Request{Input: []string{aIResponse}}, vectorChannel)
	vectorResponse := <-vectorChannel
	if len(vectorResponse.Data) == 0 || len(vectorResponse.Data[0].Embedding) == 0 {
		fmt.Printf("No embedding vector received from ai for %v\n", query)
		m.noDataFoundUI(sse)
		return
	}
	pineconeChannel := make(chan []string)
	go m.pineconeClient.Query(vectorResponse.Data[0].Embedding, pineconeChannel)
	videoIds := <-pineconeChannel
	if len(videoIds) == 0 {
		m.noDataFoundUI(sse)
		return
	}
	dbChannel := make(chan []models.VideoResponse)
	go m.dbService.FilterVideos(sse.Context(), videoIds, dbChannel)
	videos := <-dbChannel

	if len(videos) == 0 {
		m.noDataFoundUI(sse)
		return
	}
	m.loadVideosWithOffsetUI(models.LongSSEData{Data: videos, OffsetVal: 0, SearchTxt: query}, false, sse)
	m.removeLoadMoreUI(sse)
}
func (m *MainHandler) landingPageUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchElementTempl(components.LandingMainWithoutLoad(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.AddVideoButton(), datastar.WithUseViewTransitions(true))

	m.loadVideosWithOffsetUI(models.LongSSEData{Data: sseData.Data, OffsetVal: 0}, false, sse)
	if strings.TrimSpace(sseData.SearchTxt) != "" {
		m.removeLoadMoreUI(sse)
	}
}
func (m *MainHandler) HandleSearch(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.UISignals
	datastar.ReadSignals(request, &clientSignal)
	query := strings.TrimSpace(clientSignal.SearchTxt)
	// fmt.Printf("Search query received: %v\n", query)
	if query == "" {
		videos := m.helperService.GetFirstSetOfVideos(request.Context())
		if sessionSseChannel, sidExists := m.sidMap.Load(clientSignal.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.CLEAR_SEARCH_FUNCTIONALITY,
				Data:             videos,
			}
			return
		}
	}
	m.searchVideosAndSendDataToChannel(clientSignal, request.Context())

}

func (m *MainHandler) noDataFoundUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.NoDataFound("No technology videos found matching your search."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
}
func (m *MainHandler) invalidSearchUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.NoDataFound("Looks like your search isn’t technology-related. Please try a tech-related query."), datastar.WithSelector("section"), datastar.WithModeInner(), datastar.WithUseViewTransitions(true))
}

func (m *MainHandler) removeLoadMoreUI(sse *datastar.ServerSentEventGenerator) {
	sse.ExecuteScript("document.getElementById('loadMore')?.remove();", datastar.WithExecuteScriptAutoRemove(true))
}
func (m *MainHandler) searchVideosAndSendDataToChannel(data models.UISignals, ctxt context.Context) {
	// fmt.Printf("searching for '%v' \n", data.SearchTxt)
	decisionChannel := make(chan bool)
	go m.openRouterClient.VerifyTechnologyTopicSearch(data.SearchTxt, decisionChannel)
	isTechnology := <-decisionChannel

	if !isTechnology {
		fmt.Printf("Query not related to technology topics: %v\n", data.SearchTxt)
		if sessionSseChannel, sidExists := m.sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.INVALID_SEARCH_FUNCTIONALITY,
			}
		}
		return
	}
	aIResponseChannel := make(chan string)
	go m.openRouterClient.OptimizeQueryForSearch(data.SearchTxt, aIResponseChannel)
	aIResponse := <-aIResponseChannel
	if strings.TrimSpace(aIResponse) == "" {
		aIResponse = data.SearchTxt
	}
	// fmt.Printf("response for optimizequery %v\n", aIResponse)
	vectorChannel := make(chan voyage.Response)
	go m.voyageClient.CallEmbedding(voyage.Request{Input: []string{aIResponse}}, vectorChannel)
	vectorResponse := <-vectorChannel
	if len(vectorResponse.Data) == 0 || len(vectorResponse.Data[0].Embedding) == 0 {
		fmt.Printf("No embedding vector received from ai for %v\n", data.SearchTxt)
		if sessionSseChannel, sidExists := m.sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	pineconeChannel := make(chan []string)
	go m.pineconeClient.Query(vectorResponse.Data[0].Embedding, pineconeChannel)
	videoIds := <-pineconeChannel
	if len(videoIds) == 0 {
		if sessionSseChannel, sidExists := m.sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	dbChannel := make(chan []models.VideoResponse)
	go m.dbService.FilterVideos(ctxt, videoIds, dbChannel)
	videos := <-dbChannel
	if len(videos) == 0 {
		if sessionSseChannel, sidExists := m.sidMap.Load(data.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.NO_DATA_FOUND_FUNCTIONALITY,
			}
		}
		return
	}
	if sessionSseChannel, sidExists := m.sidMap.Load(data.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.SEARCH_FUNCTIONALITY,
			SearchTxt:        data.SearchTxt,
			Data:             videos,
		}
	}
}

func (m *MainHandler) HandleAddPage(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if uiSignals.IdToken != "" {
		channel := make(chan bool)
		go m.dbService.VerifyIdToken(request.Context(), uiSignals.IdToken, channel)
		isValidToken := <-channel
		if isValidToken {
			if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
				sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
					FunctionalityVal: models.ADD_PAGE_UI,
				}
			}
			return
		}
	}
	http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
}
func (m *MainHandler) addPageUi(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.AddVideo(), datastar.WithSelector("main"), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
	sse.PatchElementTempl(components.HomeButton(), datastar.WithUseViewTransitions(true))
}
func (m *MainHandler) HandleTags(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.TAGS_UI,
			Tags:             uiSignals.Tags,
			UserAgent:        userAgent,
		}
	}
}
func (m *MainHandler) tagsUI(sse *datastar.ServerSentEventGenerator, data models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(data.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.TagsList(data.Tags), datastar.WithUseViewTransitions(useViewTransition))
}

func (m *MainHandler) HandleAddVideo(responseWriter http.ResponseWriter, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	uiSignals.Title = strings.TrimSpace(uiSignals.Title)
	uiSignals.Subtitle = strings.TrimSpace(uiSignals.Subtitle)
	uiSignals.VideoId = strings.TrimSpace(uiSignals.VideoId)
	uiSignals.Transcript = strings.TrimSpace(uiSignals.Transcript)

	if uiSignals.IdToken != "" {
		verifyTokenChannel := make(chan bool)
		go m.dbService.VerifyIdToken(request.Context(), uiSignals.IdToken, verifyTokenChannel)
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
				go m.yTService.GetYTVideoResponse(uiSignals.VideoId, ytVideoSearchChannel)
				ytResponse = <-ytVideoSearchChannel
				if len(ytResponse.Items) == 0 || ytResponse.Items[0].Id == "" {
					errorMessages = append(errorMessages, "Please enter a valid YouTube video ID.")
					errorSignals += "videoIdError:true,"
				}
			}
			// sse := datastar.NewSSE(responseWriter, request)
			if len(errorMessages) > 0 {
				if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
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
			go m.dbService.UpsertVideo(uiSignals, saveToDbChannel)
			success := <-saveToDbChannel
			if success {
				if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal:      models.ADD_VIDEO_SUCCESS_FUNCTIONALITY,
						AddVideoErrorMessages: []string{},
						AddVideoErrorsignals:  "",
						UserAgent:             userAgent,
					}
				}
				deleteDocIdAndVideoIdNotMatchChannel := make(chan bool)
				go m.dbService.CheckAndDeleteIfDocIdAndVideoIdAreNotSame(uiSignals.VideoId, deleteDocIdAndVideoIdNotMatchChannel)

				dataToVectorize := models.VideoResponse{
					Title:      uiSignals.Title,
					Subtitle:   uiSignals.Subtitle,
					Tags:       trimmedTags,
					Transcript: uiSignals.Transcript,
				}
				dataToVectorize = m.helperService.ConstructTextToVectorize(dataToVectorize, ytResponse.Items[0].Snippet.Description)
				vectorChannel := make(chan voyage.Response)
				go m.voyageClient.CallEmbedding(voyage.Request{Input: []string{dataToVectorize.TextToVectorize}}, vectorChannel)
				vectorData := <-vectorChannel
				if len(vectorData.Data) != 0 && len(vectorData.Data[0].Embedding) != 0 {
					upsertPineconeChannel := make(chan int)
					go m.pineconeClient.Upsert(uiSignals.VideoId, vectorData.Data[0].Embedding, upsertPineconeChannel)
					<-upsertPineconeChannel
					// fmt.Printf("Text vectorized and upserted to Pinecone: %v\n", dataToVectorize.TextToVectorize)
				}
				<-deleteDocIdAndVideoIdNotMatchChannel
			} else if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
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

func (m *MainHandler) addVideoValidationErrorUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoValidationError(sseData.AddVideoErrorMessages), datastar.WithUseViewTransitions(useViewTransition))
	sse.PatchSignals([]byte(`{` + sseData.AddVideoErrorsignals + `}`))
}
func (m *MainHandler) addVideoSuccessUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoSuccessResult(), datastar.WithUseViewTransitions(useViewTransition))
	sse.PatchSignals([]byte(`{videoId:'',title:'',subtitle:'',tags:[],rank:1,transcript:''}`))
	sse.PatchElementTempl(components.TagsList([]string{}), datastar.WithUseViewTransitions(useViewTransition))
}
func (m *MainHandler) addVideoErrorUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	useViewTransition := true
	if strings.Contains(strings.ToLower(sseData.UserAgent), "mobile") {
		useViewTransition = false
	}
	sse.PatchElementTempl(components.AddVideoErrorResult(), datastar.WithUseViewTransitions(useViewTransition))

}
func (m *MainHandler) HandleDeleteVideo(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if uiSignals.IdToken != "" {
		verifyTokenChannel := make(chan bool)
		go m.dbService.VerifyIdToken(request.Context(), uiSignals.IdToken, verifyTokenChannel)
		isValidToken := <-verifyTokenChannel
		if isValidToken {
			if uiSignals.VideoToDelete == "" {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
			deleteVideoChannel := make(chan bool)
			go m.dbService.DeleteVideo(uiSignals.VideoToDelete, deleteVideoChannel)
			success := <-deleteVideoChannel
			if success {
				if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal: models.DELETE_VIDEO_SUCCESS_FUNCTIONALITY,
						SearchTxt:        uiSignals.SearchTxt,
						VideoDeleted:     uiSignals.VideoToDelete,
					}
				}
				deletePineconeRecordChannel := make(chan bool)
				go m.pineconeClient.DeleteRecord(uiSignals.VideoToDelete, deletePineconeRecordChannel)
				<-deletePineconeRecordChannel
			} else if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
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

func (m *MainHandler) deleteVideoSuccessUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchElementTempl(components.EmptyDeleteVideoResult(), datastar.WithUseViewTransitions(true))
	sse.PatchSignals([]byte(`{videoToDelete:'',showDeleteConfirm:false}`))
	time.Sleep(200 * time.Millisecond) //wait for UI animation
	sse.RemoveElement("#playerContainer_"+sseData.VideoDeleted+"_"+sseData.SearchTxt, datastar.WithUseViewTransitions(true))
}

func (m *MainHandler) deleteVideoErrorUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.DeleteVideoErrorResult(), datastar.WithUseViewTransitions(false))
}
func (m *MainHandler) HandleLoadQuiz(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	if strings.TrimSpace(uiSignals.QuizVideoId) == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.LOAD_QUIZ_UI_FUNCTIONALITY,
			QuizVideoId:      uiSignals.QuizVideoId,
		}
	}
}
func (m *MainHandler) loadQuizUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	sse.PatchSignals([]byte(`{_loadingQuiz:false,_showQuiz:true}`))
	dbChannel := make(chan []models.VideoResponse)
	go m.dbService.FilterVideos(sse.Context(), []string{sseData.QuizVideoId}, dbChannel)
	quizVideos := <-dbChannel
	transcript := ""
	if len(quizVideos) > 0 && strings.TrimSpace(quizVideos[0].Transcript) != "" {
		transcript = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(quizVideos[0].Transcript, "'", "\\'"), "\n", "\\n"))
	}
	// fmt.Printf("Transcript for quiz generation: %v\n", transcript[0:100])
	sse.PatchElementTempl(components.CreateQuizForm(),
		datastar.WithModeInner(), datastar.WithSelector("#quizDialog"))
	sse.PatchSignals([]byte(`{transcript:'` + transcript + `'}`))
}
func (m *MainHandler) quizGeneratingUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.QuizGenerating(), datastar.WithSelector("#quizDialog"), datastar.WithModeInner())
}
func (m *MainHandler) HandleQuizGenerationVerifyAnswerAndPrevNext(responseWriter http.ResponseWriter, request *http.Request) {
	var uiSignals models.UISignals
	datastar.ReadSignals(request, &uiSignals)
	transcript := strings.TrimSpace(uiSignals.Transcript)
	if transcript == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	// fmt.Printf("Transcript %v\n",transcript)
	if quizMapItem, quizMapExists := m.quizMap.Load(uiSignals.Sid); quizMapExists {
		quizResponse := quizMapItem.(openRouter.QuizResponse)
		if quizResponse.VideoId == uiSignals.QuizVideoId {
			if uiSignals.QuizIndex < 0 || uiSignals.QuizIndex >= len(quizResponse.Questions) {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
			var evaluationAnswer openRouter.AnswerEvaluationResponse
			if uiSignals.VerifyAnswer {
				if len(strings.TrimSpace(uiSignals.Answer)) < 25 {
					http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
					return
				} else {
					answerEvaluationChannel := make(chan openRouter.AnswerEvaluationResponse)
					go m.openRouterClient.VerifyQuizAnswer(uiSignals.Answer, quizResponse, uiSignals.QuizIndex, answerEvaluationChannel)
					evaluationAnswer = <-answerEvaluationChannel
				}
			}
			if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
				switch {
				case uiSignals.VerifyAnswer && evaluationAnswer.FluencyScore == 0 && evaluationAnswer.AccuracyScore == 0:
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal: models.QUIZ_VERIFY_ANSWER_ERROR,
						QuizIndex:        uiSignals.QuizIndex,
						Sid:              uiSignals.Sid,
					}

				case uiSignals.VerifyAnswer:
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal: models.QUIZ_VERIFY_ANSWER_FUNCTIONALITY,
						QuizIndex:        uiSignals.QuizIndex,
						Sid:              uiSignals.Sid,
						Answer: models.EvaluationAnswer{
							FluencyScore:         evaluationAnswer.FluencyScore,
							AccuracyScore:        evaluationAnswer.AccuracyScore,
							FeedbackTip:          evaluationAnswer.FeedbackTip,
							ImprovedSpokenAnswer: evaluationAnswer.ImprovedSpokenAnswer,
							UsedKeywords:         evaluationAnswer.UsedKeywords,
							MissingKeywords:      evaluationAnswer.MissingKeywords,
							IsTechnicallyCorrect: evaluationAnswer.IsTechnicallyCorrect,
						},
					}
				default:
					sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
						FunctionalityVal: models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY,
						QuizIndex:        uiSignals.QuizIndex,
						Sid:              uiSignals.Sid,
					}
				}
			}
			return
		}
	}
	if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
		sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
			FunctionalityVal: models.QUIZ_GENERATING_FUNCTIONALITY,
		}
	}
	openRouterChannel := make(chan openRouter.QuizResponse)
	go m.openRouterClient.GenerateQuiz(uiSignals, openRouterChannel)

	updateTranscriptChannel := make(chan bool)
	go m.dbService.UpdateTranscriptForQuiz(uiSignals, updateTranscriptChannel)

	quizResponse := <-openRouterChannel
	if len(quizResponse.Questions) > 0 {
		quizResponse.VideoId = uiSignals.QuizVideoId
		m.quizMap.Store(uiSignals.Sid, quizResponse)

		if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.QUIZ_AND_PREV_NEXT_FUNCTIONALITY,
				QuizIndex:        0,
				Sid:              uiSignals.Sid,
			}
		}
	} else {
		if sessionSseChannel, sidExists := m.sidMap.Load(uiSignals.Sid); sidExists {
			sessionSseChannel.(chan models.LongSSEData) <- models.LongSSEData{
				FunctionalityVal: models.QUIZ_GENERATION_ERROR_FUNCTIONALTIY,
			}
		}
	}
	<-updateTranscriptChannel
}

func (m *MainHandler) quizQuestionUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	quizResponse, _ := m.quizMap.Load(sseData.Sid)
	sse.PatchElementTempl(components.QuizQuestion(quizResponse.(openRouter.QuizResponse), sseData.QuizIndex),
		datastar.WithSelector("#quizDialog"),
		datastar.WithModeInner())
}
func (m *MainHandler) quizAnswerVerificationUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	quizResponse, _ := m.quizMap.Load(sseData.Sid)
	sse.PatchElementTempl(components.ResultAndPrevNextQuestion(sseData.Answer, quizResponse.(openRouter.QuizResponse), sseData.QuizIndex),
		datastar.WithModeOuter())
	sse.PatchSignals([]byte(`{verifyAnswer:false}`))
}
func (m *MainHandler) quizGenerationErrorUI(sse *datastar.ServerSentEventGenerator) {
	sse.PatchElementTempl(components.QuizGenerationError(),
		datastar.WithSelector("#quizDialog"),
		datastar.WithModeInner())
}

func (m *MainHandler) quizAnswerVerificationErrorUI(sse *datastar.ServerSentEventGenerator, sseData models.LongSSEData) {
	quizResponse, _ := m.quizMap.Load(sseData.Sid)
	sse.PatchElementTempl(components.AnswerEvaluationError(sseData.QuizIndex, quizResponse.(openRouter.QuizResponse)),
		datastar.WithModeOuter())
}
