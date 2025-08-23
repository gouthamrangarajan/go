package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"datastar-claude-chat/models"
	"encoding/base64"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ImgRegex = regexp.MustCompile(`^data:(image/(png|jpeg|jpg|webp));base64,([A-Za-z0-9+/=]+)$`)

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
func InsertChatSessionViaChannel(userId string, title string, webSearch bool) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	defer close(insertSessionChannel)
	go InsertChatSession(userId, title, webSearch, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}
func GenerateClaudeRequest(userId string, sessionId int, prompt string, promptImgData string, searchWeb bool) (models.ClaudeRequest, string) {
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
	temperatureStr := os.Getenv("CLAUDE_TEMPERATURE")
	temperature, err := strconv.ParseFloat(temperatureStr, 32)
	if err != nil {
		temperature = 0.5
	}
	claudeRequest := models.ClaudeRequest{
		Model:       os.Getenv("CLAUDE_MODEL"),
		MaxToken:    maxToken,
		Stream:      true,
		Temperature: float32(temperature),
	}
	claudeRequest.Messages = make([]models.ClaudeRequestMessage, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.Message) != "" {
			if conversation.ImgData != "" {
				matches := ImgRegex.FindStringSubmatch(conversation.ImgData)
				if len(matches) > 3 {
					contentWithImage := []models.ClaudeRequestImageContent{}
					contentWithImage = append(contentWithImage, models.ClaudeRequestImageContent{
						Type: "image",
						Source: models.ClaudeRequestImageContentSource{
							Type:      "base64",
							MediaType: matches[1],
							Data:      matches[3],
						},
					})
					contentWithImage = append(contentWithImage, models.ClaudeRequestImageContent{
						Type: "text",
						Text: conversation.Message,
					})
					claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
						Role:             conversation.Sender,
						ContentWithImage: contentWithImage,
					})
				}
			} else {
				claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
					Role:    conversation.Sender,
					Content: conversation.Message,
				})
			}
		}
	}
	if promptImgData != "" {
		matches := ImgRegex.FindStringSubmatch(promptImgData)
		if len(matches) > 3 {
			contentWithImage := []models.ClaudeRequestImageContent{}
			contentWithImage = append(contentWithImage, models.ClaudeRequestImageContent{
				Type: "image",
				Source: models.ClaudeRequestImageContentSource{
					Type:      "base64",
					MediaType: matches[1],
					Data:      matches[3],
				},
			})
			contentWithImage = append(contentWithImage, models.ClaudeRequestImageContent{
				Type: "text",
				Text: prompt,
			})
			claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
				Role:             "user",
				ContentWithImage: contentWithImage,
			})
		}
	} else {
		claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
			Role:    "user",
			Content: prompt,
		})
	}
	if searchWeb {
		max_uses_str := os.Getenv("CALUDE_WEB_TOOL_MAX_USES")
		max_uses, err := strconv.Atoi(max_uses_str)
		if err != nil {
			max_uses = 3
		}
		claudeRequest.Tools = []models.ClaudeRequestTools{}
		claudeRequest.Tools = append(claudeRequest.Tools, models.ClaudeRequestTools{
			Type:    os.Getenv("CLAUDE_WEB_TOOL_TYPE"),
			Name:    os.Getenv("CLAUDE_WEB_TOOL_NAME"),
			MaxUses: max_uses,
		})
	}
	return claudeRequest, errToRet
}

func SendErrorMessageToUI(sse *datastar.ServerSentEventGenerator, message string) {
	sse.PatchSignals([]byte(`{showErrorMessage:true,errorMessage:'` + message + `'}`))
	time.Sleep(3000 * time.Millisecond)
	sse.PatchSignals([]byte("{showErrorMessage:false}"))
}
