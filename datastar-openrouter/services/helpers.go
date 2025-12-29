package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"datastar-openrouter/models"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userId"

func GenerateSignedStrForCookie(model models.UICookie) string {
	cookieSecret := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(model.Name))
	mac.Write([]byte(model.Value))
	signature := mac.Sum(nil)
	cookieValueSignedBytes := append(signature, []byte(model.Value)...)
	cookieValueSignedStr := base64.URLEncoding.EncodeToString(cookieValueSignedBytes)
	return cookieValueSignedStr
}
func GetUserIdFromRequest(request *http.Request) string {
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

func GetChatSessionsViaChannel(userId string) []models.ChatSession {
	sessionChannel := make(chan []models.ChatSession)
	defer close(sessionChannel)
	go GetChatSessions(userId, sessionChannel)
	sessions := <-sessionChannel
	return sessions
}
func InsertChatSessionViaChannel(userId string, data models.ChatSession) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	defer close(insertSessionChannel)
	go InsertChatSession(userId, data, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}

func GenerateOpenRouterRequest(userId string, request models.ClientSignals) (models.OpenRouterRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, request.SessionId, conversationsChannel)
	conversations := <-conversationsChannel

	openRouterRequest := models.OpenRouterRequest{
		Stream: true,
		Model:  "openrouter/auto:nitro",
	}
	openRouterRequest.Messages = make([]models.OpenRouterRequestMessage, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.Content) != "" {
			openRouterRequest.Messages = append(openRouterRequest.Messages, models.OpenRouterRequestMessage{
				Role:    conversation.Role,
				Content: conversation.Content,
			})

		}
	}

	// fmt.Printf("Generated OpenRouter Request: %+v\n", openRouterRequest)
	return openRouterRequest, errToRet
}
