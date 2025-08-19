package main

import (
	"datastar-claude-chat/components"
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sessionId := 0
	sessionIdStr := chi.URLParam(request, "sessionId")
	sessionId, err := strconv.Atoi(sessionIdStr)
	if err != nil {
		sessionId = 0
	}
	if sessionId == 0 {
		components.Main(0, []models.ChatConversation{}).Render(request.Context(), responseWriter)
		return
	}
	userId := request.Context().Value(services.UserIDKey).(string)

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	sessionFound := false
	for _, session := range sessions {
		if session.Id == sessionId {
			sessionFound = true
			break
		}
	}
	if !sessionFound {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}
	conversationChannel := make(chan []models.ChatConversation)
	defer close(conversationChannel)
	go services.GetChatConversations(userId, sessionId, conversationChannel)
	conversations := <-conversationChannel
	components.Main(sessionId, conversations).Render(request.Context(), responseWriter)
}

func menuDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	sessions := services.GetChatSessionsViaChannel(userId)
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.ChatSessionMenuItems(sessions), datastar.WithModeInner(), datastar.WithSelectorID("menuContainer"), datastar.WithUseViewTransitions(true))
}

func newChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	newSessionId := services.InsertChatSessionViaChannel(userId, "New Chat")
	sse := datastar.NewSSE(responseWriter, request)
	if newSessionId == 0 {
		sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'Failed to create new conversation. Please try again later.'}"))
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
		return
	}
	sse.PatchSignals([]byte("{showMenu:false}"))
	time.Sleep(200 * time.Millisecond)
	sse.ExecuteScript("window.location.href=window.location.origin+'/'+" + strconv.Itoa(newSessionId))
}
