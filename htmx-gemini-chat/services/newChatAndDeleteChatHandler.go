package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"strconv"
)

func NewChatSessionHandler(response http.ResponseWriter, request *http.Request) {
	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	newChatSessionId := InsertChatSessionViaChannel(userId, "New Chat")
	if newChatSessionId == 0 {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	component := components.NewChatSession(models.ChatSession{Id: newChatSessionId, Title: "New Chat"})
	component.Render(request.Context(), response)
}

func DeleteSessionHandler(response http.ResponseWriter, request *http.Request,
	chatSessionIdToDelete int) {

	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	chatSessionIdDataDisplayedInUIStr := request.FormValue("chatSessionId")
	chatSessionIdDataDisplayedInUI, err := strconv.Atoi(chatSessionIdDataDisplayedInUIStr)
	if err != nil {
		chatSessionIdDataDisplayedInUI = 0
	}
	if chatSessionIdToDelete <= 0 { // RG url sends non integer value
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	sessions := GetChatSessionsViaChannel(userId)

	ftedSessions := make([]models.ChatSession, 0, 1)
	for _, session := range sessions {
		if session.Id == chatSessionIdToDelete {
			ftedSessions = append(ftedSessions, session)
			break
		}
	}
	if len(ftedSessions) == 0 { //RG url sends an chatSessionId not belonging to user
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}

	channel := make(chan int)
	defer close(channel)
	go DeleteChatSession(userId, chatSessionIdToDelete, channel)
	rowsAffected := <-channel
	if rowsAffected <= 0 {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if chatSessionIdToDelete == chatSessionIdDataDisplayedInUI {
		//change UI & return
		sessions := GetChatSessionsViaChannel(userId)
		if len(sessions) > 0 {
			conversationsChannel := make(chan []models.ChatConversation)
			defer close(conversationsChannel)
			go GetChatConversations(userId, sessions[0].Id, conversationsChannel)
			conversations := <-conversationsChannel
			component := components.UIToReplaceDeleteChatSession(conversations, sessions[0].Id)
			component.Render(request.Context(), response)
		} else {
			component := components.UIToReplaceDeleteChatSession([]models.ChatConversation{}, 0)
			component.Render(request.Context(), response)
		}

		return
	}
	response.WriteHeader(http.StatusOK)
}
