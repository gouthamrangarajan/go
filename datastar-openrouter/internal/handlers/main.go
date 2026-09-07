package handlers

import (
	"bytes"
	"context"
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components"
	"datastar-openrouter/services"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

var uiSidMap sync.Map

func MainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// style := styles.Get("github") // Use the same style as in goldmark
	// formatter := html.New(html.WithClasses(true))
	// buf := bytes.Buffer{}
	// formatter.WriteCSS(&buf, style)
	// fmt.Println(buf.String())
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionId := 0
	sessionIdStr := chi.URLParam(request, "sessionId")
	searchMenuTxt := strings.TrimSpace(request.URL.Query().Get("search_menu"))
	sessionId, err := strconv.Atoi(sessionIdStr)
	if err != nil {
		sessionId = 0
	}
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == sessionId {
			selectedSession = session
			break
		}
	}
	if selectedSession.Id == 0 && sessionId != 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}

	chatConversationChannel := make(chan []models.ChatConversation)
	defer close(chatConversationChannel)
	go services.GetChatConversationsWithoutMessageAndFileData(userId, sessionId, chatConversationChannel)

	chatConversations := <-chatConversationChannel

	if request.Header.Get("Datastar-Request") == "true" {
		sessionChangeHandler(request, models.SessionChangeData{
			UserId:            userId,
			Session:           selectedSession,
			ChatConversations: chatConversations,
			SearchMenuText:    searchMenuTxt,
		})
		return
	}

	aiModelsChannel := make(chan []models.AIModel)
	defer close(aiModelsChannel)
	go services.GetAiModels(aiModelsChannel)

	if searchMenuTxt != "" {
		sessions = services.SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: searchMenuTxt,
		})
	}
	aiModels := <-aiModelsChannel

	components.Main(
		models.UIMainModel{
			Messages:         chatConversations,
			Sessions:         sessions,
			AIModels:         aiModels,
			AllowWebSearch:   selectedSession.AllowWebSearch,
			ImageGeneration:  selectedSession.ImageGeneration,
			CurrentSessionId: sessionId,
			MenuSearchTerm:   searchMenuTxt,
		}).Render(request.Context(), responseWriter)
}

func sessionChangeHandler(request *http.Request, data models.SessionChangeData) {
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	clientSignal.SessionId = data.Session.Id
	userSessionKey := services.GenerateUserSessionKey(data.UserId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		dataBuffer := new(bytes.Buffer)
		components.Section(data.ChatConversations).Render(context.Background(), dataBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           dataBuffer.String(),
			UseViewTransition: false,
			Mode:              datastar.WithModeOuter(),
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content: `{sessionId:` + strconv.Itoa(data.Session.Id) + `,webSearch:` + strconv.FormatBool(data.Session.AllowWebSearch) +
				`,imageGeneration:` + strconv.FormatBool(data.Session.ImageGeneration) +
				`,messageIdToFetchImage:0,showMenu:false,showErrorMessage:false,showDeleteModal:false
					,sessionIdToDelete:0}`,
			UseViewTransition: true,
		}

		urlToReplace := `/` + strconv.Itoa(data.Session.Id)
		if strings.TrimSpace(data.SearchMenuText) != "" {
			urlToReplace += "?search_menu=" + data.SearchMenuText
		}
		// fmt.Printf("Replacing URL with: %s\n", urlToReplace)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `window.history.replaceState({},"","` + urlToReplace + `")`,
			IsScript: true,
		}

		//make sure all before data are flushed before sending the markdown to html data
		// userSession.(chan models.LongSSEData) <- models.LongSSEData{
		// 	SendHeartBeat: true,
		// }

	}
	go sendConversationsMarkdown(clientSignal, data.UserId)
}
