package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"datastar-openrouter/models"
	"encoding/base64"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ImgRegex = regexp.MustCompile(`^data:(image/(png|jpeg|jpg|webp|gif));base64,([A-Za-z0-9+/=]+)$`)
var PdfRegex = regexp.MustCompile(`^data:application/pdf;base64,([A-Za-z0-9+/=]+)$`)

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

func GenerateOpenRouterRequest(userId string, clientSignal models.ClientSignals) (models.OpenRouterRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel
	if strings.TrimSpace(clientSignal.ModelId) == "" {
		clientSignal.ModelId = os.Getenv("DEFAULT_MODEL_ID")
	} else {
		clientSignal.ModelId += ":nitro"
	}

	openRouterRequest := models.OpenRouterRequest{
		Stream: true,
		Model:  clientSignal.ModelId,
	}
	openRouterRequest.Messages = make([]models.OpenRouterRequestMessage, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.FileData) != "" {
			messageToAppend := models.OpenRouterRequestMessage{
				Role: conversation.Role,
			}
			messageToAppend.ContentWithFileData = append(messageToAppend.ContentWithFileData,
				models.OpenRouterRequestMessageContentWithFileData{
					Type: "text",
					Text: conversation.Content,
				})
			var contentWithFileData models.OpenRouterRequestMessageContentWithFileData
			if ImgRegex.MatchString(conversation.FileData) {
				contentWithFileData = models.OpenRouterRequestMessageContentWithFileData{
					Type: "image_url",
					ImageUrl: struct {
						Url string `json:"url,omitempty"`
					}{Url: conversation.FileData},
				}
			} else if PdfRegex.MatchString(conversation.FileData) {
				contentWithFileData = models.OpenRouterRequestMessageContentWithFileData{
					Type: "file",
					File: struct {
						Name string `json:"filename,omitempty"`
						Data string `json:"file_data,omitempty"`
					}{
						Name: conversation.FileName,
						Data: conversation.FileData,
					},
				}
			}
			messageToAppend.ContentWithFileData = append(messageToAppend.ContentWithFileData, contentWithFileData)
			openRouterRequest.Messages = append(openRouterRequest.Messages, messageToAppend)
		} else if strings.TrimSpace(conversation.Content) != "" {
			openRouterRequest.Messages = append(openRouterRequest.Messages, models.OpenRouterRequestMessage{
				Role:    conversation.Role,
				Content: conversation.Content,
			})

		}
	}

	// fmt.Printf("Generated OpenRouter Request: %+v\n", openRouterRequest)
	return openRouterRequest, errToRet
}
func SendErrorMessageToUI(sse *datastar.ServerSentEventGenerator, message string) {
	sse.PatchSignals([]byte(`{showErrorMessage:true,errorMessage:'` + message + `'}`))
	time.Sleep(3000 * time.Millisecond)
	sse.PatchSignals([]byte("{showErrorMessage:false}"))
}
