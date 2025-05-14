package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"htmx-gemini-chat/models"
	"net/http"
	"os"
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

func GenerateGeminiRequest(userId string, sessionId int, prompt string, imgBase64 string) (models.GeminiRequest, string) {
	err := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, sessionId, conversationsChannel)
	conversations := <-conversationsChannel
	geminiRequest := models.GeminiRequest{}
	geminiRequest.Contents = make([]models.GeminiRequestContent, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.Message) != "" {
			if conversation.ImgData != "" {
				messageToGeminiRequestContent := models.GeminiRequestContent{
					Role: conversation.Sender,
					Parts: append(make([]models.GeminiRequestParts, 0, 2), models.GeminiRequestParts{
						Text: &conversation.Message,
					}),
				}
				matches := imgRegex.FindStringSubmatch(conversation.ImgData)
				if len(matches) > 3 {
					messageToGeminiRequestContent.Parts = append(messageToGeminiRequestContent.Parts, models.GeminiRequestParts{
						ImgData: &models.GeminiRequestImageData{
							MimeType: matches[1],
							Data:     matches[3],
						},
					})
				}
				geminiRequest.Contents = append(geminiRequest.Contents, messageToGeminiRequestContent)

			} else {
				geminiRequest.Contents = append(geminiRequest.Contents, models.GeminiRequestContent{
					Role: conversation.Sender,
					Parts: append(make([]models.GeminiRequestParts, 0, 1), models.GeminiRequestParts{
						Text: &conversation.Message,
					}),
				})
			}
		}
	}
	partsCapacityForPrompt := 1
	if imgBase64 != "" {
		partsCapacityForPrompt = 2
	}
	promptToGeminiRequestContent := models.GeminiRequestContent{
		Role: "user",
		Parts: append(make([]models.GeminiRequestParts, 0, partsCapacityForPrompt), models.GeminiRequestParts{
			Text: &prompt,
		}),
	}
	if imgBase64 != "" {
		matches := imgRegex.FindStringSubmatch(imgBase64)
		if len(matches) < 4 {
			err = "Invalid Image data"
		} else {
			decoded, decodeErr := base64.StdEncoding.DecodeString(matches[3])
			if decodeErr != nil || len(decoded) > 1024*1024 {
				err = "Invalid Image data"
			}
		}

		if err == "" {
			promptToGeminiRequestContent.Parts = append(promptToGeminiRequestContent.Parts, models.GeminiRequestParts{
				ImgData: &models.GeminiRequestImageData{
					MimeType: matches[1],
					Data:     matches[3],
				},
			})
		}
	}
	geminiRequest.Contents = append(geminiRequest.Contents, promptToGeminiRequestContent)
	return geminiRequest, err
}
