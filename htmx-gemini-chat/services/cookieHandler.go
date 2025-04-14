package services

import (
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

func CookieHandlerToMainPage(response http.ResponseWriter, request *http.Request) {
	var id string
	cookie, err := request.Cookie("id")
	if err != nil {
		id = uuid.New().String()
		secure := true
		if os.Getenv("ENV") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "id",
			Value:    id,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(response, &cookie)
		userChannel := make(chan int)
		defer close(userChannel)
		go InsertUser(id, userChannel)
		<-userChannel
	} else {
		id = cookie.Value
	}
	sessionChannel := make(chan []models.ChatSession)
	defer close(sessionChannel)
	go GetChatSessions(id, sessionChannel)
	sessions := <-sessionChannel
	conversations := []models.ChatConversation{}
	if len(sessions) > 0 {
		conversationsChannel := make(chan []models.ChatConversation)
		defer close(conversationsChannel)
		go GetChatConversations(id, sessions[0].Id, conversationsChannel)
		conversations = <-conversationsChannel
	}
	component := components.Main(conversations)
	component.Render(request.Context(), response)
}

func CookieHandlerToPromptHandler(response http.ResponseWriter, request *http.Request) {
	var id string
	cookie, err := request.Cookie("id")
	if err != nil {
		response.WriteHeader(401)
		return
	}
	id = cookie.Value
	promptHandler(response, request, id)
}
