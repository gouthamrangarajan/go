package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"datastar-claude-chat/models"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userId"

func GenerateSignedStrForCookie(name string, val string) string {
	cookieSecret := os.Getenv("COOKIE_SECRET")
	mac := hmac.New(sha256.New, []byte(cookieSecret))
	mac.Write([]byte(name))
	mac.Write([]byte(val))
	signature := mac.Sum(nil)
	cookieValueSignedBytes := append(signature, []byte(val)...)
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
func InsertChatSessionViaChannel(userId string, title string) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	defer close(insertSessionChannel)
	go InsertChatSession(userId, title, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}
func GenerateClaudeRequest(userId string, sessionId int, prompt string) (models.ClaudeRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, sessionId, conversationsChannel)
	conversations := <-conversationsChannel

	maxTokenStr := os.Getenv("CLAUDE_MAX_TOKEN")
	maxToken, err := strconv.Atoi(maxTokenStr)
	if err != nil {
		maxToken = 8000
	}
	claudeRequest := models.ClaudeRequest{
		Model:    os.Getenv("CLAUDE_MODEL"),
		MaxToken: maxToken,
		Stream:   true,
	}
	claudeRequest.Messages = make([]models.ClaudeRequestMessage, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.Message) != "" {
			if conversation.ImgData != "" {

			} else {
				claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
					Role:    conversation.Sender,
					Content: conversation.Message,
				})
			}
		}
	}
	claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
		Role:    "user",
		Content: prompt,
	})
	return claudeRequest, errToRet
}
