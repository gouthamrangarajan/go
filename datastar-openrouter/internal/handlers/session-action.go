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

	"github.com/starfederation/datastar-go/datastar"
)

func NewChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)

	insertChatSessionChannel := make(chan int)
	defer close(insertChatSessionChannel)
	newSession := models.ChatSession{Title: "New Chat"}
	go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
	newSession.Id = <-insertChatSessionChannel

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		if newSession.Id == 0 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: `Failed to create new chat session. Please try again later.`,
				IsError: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `{pageLoading:false}`,
				IsSignal: true,
			}
			return
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content: `{sessionId:` + strconv.Itoa(newSession.Id) + `,webSearch:false,imageGeneration:false,
				messageIdToFetchImage:0,showMenu:false,showErrorMessage:false,showDeleteModal:false,
				sessionIdToDelete:0}`,
			UseViewTransition: true,
		}
		sectionComponentBuffer := new(bytes.Buffer)
		components.Section([]models.ChatConversation{}).Render(context.Background(), sectionComponentBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           sectionComponentBuffer.String(),
			Selector:          "section",
			UseViewTransition: true,
			Mode:              datastar.WithModeOuter(),
		}
		urlToReplace := `/` + strconv.Itoa(newSession.Id)
		if strings.TrimSpace(clientSignal.SearchMenu) != "" {
			urlToReplace += "?search_menu=" + clientSignal.SearchMenu
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `window.history.replaceState({},"","` + urlToReplace + `")`,
			IsScript: true,
		}
		menuItemBuffer := new(bytes.Buffer)
		components.MenuItem(newSession, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           menuItemBuffer.String(),
			Selector:          "#menu",
			UseViewTransition: true,
			Mode:              datastar.WithModeAppend(),
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `{pageLoading:false}`,
			IsSignal: true,
		}
	}
	if newSession.Id != 0 {
		embeddingChannel := make(chan models.VoyageEmbeddingResponse)
		defer close(embeddingChannel)
		embeddingRequest := models.VoyageEmbeddingRequest{
			Input: []string{newSession.Title},
		}
		go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
		embeddingResponse := <-embeddingChannel
		if len(embeddingResponse.Data) > 0 {
			updateTitleVectorChannel := make(chan int)
			defer close(updateTitleVectorChannel)
			go services.UpdateChatSessionTitleVector(newSession.Id, embeddingResponse.Data[0].Embedding, updateTitleVectorChannel)
			<-updateTitleVectorChannel
		}
	}

}

func DeleteSessionHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)

	if clientSignal.SessionIdToDelete == 0 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == clientSignal.SessionIdToDelete {
			selectedSession = session
			break
		}
	}
	if selectedSession.Id == 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}
	deleteSessionChannel := make(chan int)
	defer close(deleteSessionChannel)
	go services.DeleteChatSession(userId, clientSignal.SessionIdToDelete, deleteSessionChannel)

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	userSession, userSessionExists := uiSidMap.Load(userSessionKey)

	if <-deleteSessionChannel == 0 {
		if userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: `Failed to delete chat session. Please try again later.`,
				IsError: true,
			}
		}
		return
	}
	if userSessionExists {
		if clientSignal.SessionIdToDelete == clientSignal.SessionId {
			componentBuffer := new(bytes.Buffer)
			components.Section([]models.ChatConversation{}).Render(context.Background(), componentBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           componentBuffer.String(),
				Selector:          "section",
				UseViewTransition: true,
				Mode:              datastar.WithModeOuter(),
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `{sessionId:0,webSearch:false}`,
				IsSignal: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `window.history.replaceState({},'','/')`,
				IsScript: true,
			}
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsRemove:          true,
			Selector:          "#menuItem_" + strconv.Itoa(clientSignal.SessionIdToDelete),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content:  `{showDeleteModal:false}`}
	}
}

func SearchSessionHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	userExistsChannel := make(chan bool)
	defer close(userExistsChannel)
	go services.CheckUserExistsInTable(userId, userExistsChannel)
	if !<-userExistsChannel {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	// sse := datastar.NewSSE(responseWriter, request)
	sessions := []models.ChatSession{}
	clientSignal.SearchMenu = strings.TrimSpace(clientSignal.SearchMenu)
	if clientSignal.SearchMenu == "" {
		sessionsChannel := make(chan []models.ChatSession)
		defer close(sessionsChannel)
		go services.GetChatSessions(userId, sessionsChannel)
		sessions = <-sessionsChannel
	} else {
		sessions = services.SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: clientSignal.SearchMenu})
	}

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		componentBuffer := new(bytes.Buffer)
		components.MenuUl(sessions, clientSignal.SearchMenu).Render(context.Background(), componentBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           componentBuffer.String(),
			UseViewTransition: true,
		}
		scriptToExecute := "window.history.replaceState({},'','/"
		if clientSignal.SessionId != 0 {
			scriptToExecute += strconv.Itoa(clientSignal.SessionId)
		}
		if strings.TrimSpace(clientSignal.SearchMenu) != "" {
			scriptToExecute += "?search_menu=" + clientSignal.SearchMenu
		}
		scriptToExecute += "')"
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  scriptToExecute,
			IsScript: true,
		}

	}
}
