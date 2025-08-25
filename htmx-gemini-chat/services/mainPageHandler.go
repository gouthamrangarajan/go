package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"strings"
	"time"
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
	allowWebSearch := false

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
		allowWebSearch = ftedSessions[0].AllowWebSearch
		conversationsChannel := make(chan []models.ChatConversation)
		defer close(conversationsChannel)
		go GetChatConversations(userId, chatSessionId, conversationsChannel)
		conversations = <-conversationsChannel
	}
	if request.Header.Get("HX-Request") == "true" {
		time.Sleep(200 * time.Millisecond) // Simulate a delay for the sake of UX so that menu closes before the chat session is loaded
		component := components.SectionAndChatSessionIdInput(chatSessionId, conversations, allowWebSearch, true)
		component.Render(request.Context(), response)
	} else {
		component := components.Main(conversations, sessions, chatSessionId, allowWebSearch)
		component.Render(request.Context(), response)
	}
}

func SearchMenuHandler(response http.ResponseWriter, request *http.Request) {
	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var sessions []models.ChatSession
	srchTxt := strings.TrimSpace(request.URL.Query().Get("srchTxt"))
	if srchTxt != "" {
		embeddingRequest := GenerateGeminiEmbeddingRequest(srchTxt)
		embeddingChannel := make(chan models.GeminiEmbeddingResponse)
		defer close(embeddingChannel)
		go CallGeminiEmbedding(embeddingRequest, embeddingChannel)
		embeddingResponse := <-embeddingChannel

		chatSessionsChannel := make(chan []models.ChatSession)
		defer close(chatSessionsChannel)
		go SearchChatSessions(userId, embeddingResponse.Embedding.Values, chatSessionsChannel)
		sessions = <-chatSessionsChannel
	} else {
		sessions = GetChatSessionsViaChannel(userId)
	}
	components.MenuContainer(sessions).Render(request.Context(), response)
}
