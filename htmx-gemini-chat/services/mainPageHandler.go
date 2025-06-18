package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
)

func MainPageHandler(response http.ResponseWriter, request *http.Request, chatSessionId int) {
	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sessions := GetChatSessionsViaChannel(userId)
	conversations := []models.ChatConversation{}

	if chatSessionId < 0 { // RG url sends non integer value
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	if chatSessionId == 0 && len(sessions) > 0 {
		chatSessionId = sessions[0].Id
	}
	if chatSessionId > 0 {
		ftedSessions := make([]models.ChatSession, 0, 1)
		for _, session := range sessions {
			if session.Id == chatSessionId {
				ftedSessions = append(ftedSessions, session)
				break
			}
		}
		if len(ftedSessions) == 0 { //RG url sends an chatSessionId not belonging to user
			http.Error(response, "Unauthorized", http.StatusUnauthorized)
			return
		}
		conversationsChannel := make(chan []models.ChatConversation)
		defer close(conversationsChannel)
		go GetChatConversations(userId, chatSessionId, conversationsChannel)
		conversations = <-conversationsChannel
	}
	if request.Header.Get("HX-Request") == "true" {
		component := components.SectionAndChatSessionIdInput(chatSessionId, conversations, true)
		component.Render(request.Context(), response)
	} else {
		component := components.Main(conversations, sessions, chatSessionId)
		component.Render(request.Context(), response)
	}
}

func MarkdownSrcHandler(sessionId int, conversationId int, response http.ResponseWriter, request *http.Request) {
	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	conversationChannel := make(chan models.ChatConversation)
	defer close(conversationChannel)
	go GetChatConversation(userId, sessionId, conversationId, conversationChannel)
	conversation := <-conversationChannel
	response.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // 1 year (31536000 seconds), immutable
	response.Write([]byte(conversation.Message))
}
