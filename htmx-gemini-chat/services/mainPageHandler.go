package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"strings"
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
	imgGeneration := false

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
		imgGeneration = ftedSessions[0].ImageGeneration
		conversationsChannel := make(chan []models.ChatConversation)
		defer close(conversationsChannel)
		go GetChatConversations(userId, chatSessionId, conversationsChannel)
		conversations = <-conversationsChannel
	}
	srchTxt := strings.TrimSpace(request.URL.Query().Get("search_menu"))

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	filterOrGetSessions(struct {
		UserId      string
		MenuSrchTxt string
	}{UserId: userId, MenuSrchTxt: srchTxt}, sessionsChannel)
	sessions = <-sessionsChannel
	component := components.Main(conversations, sessions, models.Section{ChatSessionId: chatSessionId, WebSearch: allowWebSearch, ImageGeneration: imgGeneration, HelperTextShow: len(conversations) == 0, MenuSrchTxt: srchTxt})
	component.Render(request.Context(), response)

}

func SearchMenuHandler(response http.ResponseWriter, request *http.Request) {
	userId, ok := request.Context().Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	srchTxt := strings.TrimSpace(request.URL.Query().Get("srchTxt"))
	channel := make(chan []models.ChatSession)
	defer close(channel)
	filterOrGetSessions(struct {
		UserId      string
		MenuSrchTxt string
	}{UserId: userId, MenuSrchTxt: srchTxt}, channel)
	sessions := <-channel
	components.MenuContainer(sessions, srchTxt).Render(request.Context(), response)
}

func filterOrGetSessions(filterSessionModel struct {
	UserId      string
	MenuSrchTxt string
}, channel chan<- []models.ChatSession) {
	if filterSessionModel.MenuSrchTxt != "" {
		embeddingRequest := GenerateGeminiEmbeddingRequest(filterSessionModel.MenuSrchTxt)
		embeddingChannel := make(chan models.GeminiEmbeddingResponse)
		defer close(embeddingChannel)
		go CallGeminiEmbedding(embeddingRequest, embeddingChannel)
		embeddingResponse := <-embeddingChannel
		go SearchChatSessions(filterSessionModel.UserId, embeddingResponse.Embedding.Values, channel)
	} else {
		go GetChatSessions(filterSessionModel.UserId, channel)
	}
}
