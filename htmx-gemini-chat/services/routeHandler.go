package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func RouteHandlerToMainPage(response http.ResponseWriter, request *http.Request, chatSessionId int) {
	userId := getUserIdFromRequest(request)
	if userId == "" {
		userId = uuid.New().String()
		secure := true
		if os.Getenv("ENV") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "id",
			Value:    generateSignedStrForCookie("id", userId),
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
	}
	sessions := getChatSessionsViaChannel(userId)
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
	component := components.Main(conversations, sessions, chatSessionId)
	component.Render(request.Context(), response)
}

func generateSignedStrForCookie(name string, val string) string {
	cookieSecret := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(name))
	mac.Write([]byte(val))
	signature := mac.Sum(nil)
	cookieValueSignedBytes := append(signature, []byte(val)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)
	return cookieValueSignedStr
}
func getUserIdFromRequest(request *http.Request) string {
	cookieName := "id"
	cookie, err := request.Cookie("id")
	if err != nil {
		return ""
	}
	cookieVal := cookie.Value
	cookieSecret := os.Getenv("COOKIE_SECRET")
	cookieValueDecoded, err := base64.URLEncoding.DecodeString(cookieVal)
	if err != nil {
		return ""
	}
	if len(cookieValueDecoded) <= sha256.Size {
		return ""
	}
	signatureFromCookie := cookieValueDecoded[:sha256.Size]
	userIdFromCookie := cookieValueDecoded[sha256.Size:]
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(cookieName))
	mac.Write([]byte(userIdFromCookie))
	signature := mac.Sum(nil)
	if !hmac.Equal(signature, signatureFromCookie) {
		return ""
	}
	return string(userIdFromCookie)
}
func RouteHandlerToPromptHandler(response http.ResponseWriter, request *http.Request) {
	userId := getUserIdFromRequest(request)
	if userId == "" {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	promptHandler(response, request, userId)
}

func RouteHandlerToNewChatSession(response http.ResponseWriter, request *http.Request) {
	userId := getUserIdFromRequest(request)
	if userId == "" {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	newChatSessionId := insertChatSessionViaChannel(userId, "New Chat")
	if newChatSessionId == 0 {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	component := components.NewChatSession(models.ChatSession{Id: newChatSessionId, Title: "New Chat"})
	component.Render(request.Context(), response)
}

func RouteHandlerToDeleteSession(response http.ResponseWriter, request *http.Request,
	chatSessionIdToDelete int) {

	chatSessionIdDataDisplayedInUIStr := request.FormValue("chatSessionId")
	chatSessionIdDataDisplayedInUI, err := strconv.Atoi(chatSessionIdDataDisplayedInUIStr)
	if err != nil {
		chatSessionIdDataDisplayedInUI = 0
	}
	userId := getUserIdFromRequest(request)
	if userId == "" {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if chatSessionIdToDelete <= 0 { // RG url sends non integer value
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	sessions := getChatSessionsViaChannel(userId)

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
		sessions := getChatSessionsViaChannel(userId)
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
