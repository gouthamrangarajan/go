package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func CookieHandlerToMainPage(response http.ResponseWriter, request *http.Request) {
	var userId string
	cookie, err := request.Cookie("id")
	if err != nil {
		userId = uuid.New().String()
		secure := true
		if os.Getenv("ENV") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "id",
			Value:    userId,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(response, &cookie)
		userChannel := make(chan int)
		defer close(userChannel)
		go InsertUser(userId, userChannel)
		<-userChannel
	} else {
		userId = cookie.Value
	}
	sessions := getChatSessionsViaChannel(userId)
	conversations := []models.ChatConversation{}
	chatSessionId := getChatSessionIdFromUrl(request)
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
	component := components.Main(conversations, sessions, chatSessionId)
	component.Render(request.Context(), response)
}

func getChatSessionIdFromUrl(request *http.Request) int {
	chatSessionId := 0
	chatSessionIdStr := request.URL.Query().Get("id")
	if chatSessionIdStr != "" {
		val, err := strconv.Atoi(chatSessionIdStr)
		if err != nil {
			chatSessionId = -1
		} else {
			chatSessionId = val
		}
	}
	return chatSessionId
}

func CookieHandlerToPromptHandler(response http.ResponseWriter, request *http.Request) {
	var userId string
	cookie, err := request.Cookie("id")
	if err != nil {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userId = cookie.Value
	promptHandler(response, request, userId)
}

func CookieHandlerToNewChatSession(response http.ResponseWriter, request *http.Request) {
	var userId string
	cookie, err := request.Cookie("id")
	if err != nil {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userId = cookie.Value
	newChatSessionId := insertChatSessionViaChannel(userId, "New Chat")
	component := components.NewChatSession(models.ChatSession{Id: newChatSessionId, Title: "New Chat"})
	component.Render(request.Context(), response)
}
