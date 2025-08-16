package main

import (
	"datastar-claude-chat/components"
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"net/http"
)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	sessionId := 0
	if len(sessions) == 0 {
		newSessionChannel := make(chan int)
		go services.InsertChatSession(userId, "New Chat", newSessionChannel)
		sessionId = <-newSessionChannel
	} else {
		sessionId = sessions[0].Id
	}
	if sessionId == 0 {
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	conversationChannel := make(chan []models.ChatConversation)
	defer close(conversationChannel)
	go services.GetChatConversations(userId, sessionId, conversationChannel)
	conversations := <-conversationChannel
	components.Main(conversations).Render(request.Context(), responseWriter)
}
