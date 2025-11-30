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
func GenerateClaudeRequest(userId string, request models.PromptRequest) (models.ClaudeRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, request.SessionId, conversationsChannel)
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
			imgMatches := ImgRegex.FindStringSubmatch(conversation.ImgData)
			pdfMatches := PdfRegex.FindStringSubmatch(conversation.PdfData)
			if (conversation.ImgData != "" && len(imgMatches) == 4) ||
				(conversation.PdfData != "" && len(pdfMatches) == 2) {
				contentWithImageOrPdf := []models.ClaudeRequestImageOrPdfInlineContent{}
				if conversation.ImgData != "" && len(imgMatches) == 4 {
					contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
						Type: "image",
						Source: &models.ClaudeRequestImageOrPdfInlineContentSource{Type: "base64",
							MediaType: imgMatches[1],
							Data:      imgMatches[3]},
					})
				} else if conversation.PdfData != "" && len(pdfMatches) == 2 {
					contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
						Type: "document",
						Source: &models.ClaudeRequestImageOrPdfInlineContentSource{Type: "base64",
							MediaType: "application/pdf",
							Data:      pdfMatches[1]},
					})
				}
				contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
					Type: "text",
					Text: conversation.Message,
				})
				claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
					Role:                  conversation.Sender,
					ContentWithImageOrPdf: contentWithImageOrPdf,
				})
			} else {
				claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
					Role:    conversation.Sender,
					Content: conversation.Message,
				})
			}
		}
	}
	if request.FileData != "" {
		contentWithImageOrPdf := []models.ClaudeRequestImageOrPdfInlineContent{}
		if request.FileMediaType != "application/pdf" {
			contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
				Type: "image",
				Source: &models.ClaudeRequestImageOrPdfInlineContentSource{Type: "base64",
					MediaType: request.FileMediaType,
					Data:      request.FileData},
			})
		} else {
			contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
				Type: "document",
				Source: &models.ClaudeRequestImageOrPdfInlineContentSource{Type: "base64",
					MediaType: "application/pdf",
					Data:      request.FileData},
			})
		}
		contentWithImageOrPdf = append(contentWithImageOrPdf, models.ClaudeRequestImageOrPdfInlineContent{
			Type: "text",
			Text: request.Prompt,
		})
		claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
			Role:                  "user",
			ContentWithImageOrPdf: contentWithImageOrPdf,
		})

	} else {
		claudeRequest.Messages = append(claudeRequest.Messages, models.ClaudeRequestMessage{
			Role:    "user",
			Content: request.Prompt,
		})
	}
	if request.SearchWeb {
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

func CallEmbeddingAndUpdateSessionTitleVector(sessionId int, sessionTitle string, channel chan<- int) {
	embeddingChannel := make(chan models.VoyageEmbeddingResponse)
	if len(sessionTitle) > 500 {
		sessionTitle = sessionTitle[:500]
	}
	go CallVoyageEmbedding(models.VoyageEmbeddingRequest{
		Input: []string{sessionTitle},
		Model: os.Getenv("VOYAGE_EMBEDDINGS_MODEL"),
	}, embeddingChannel)
	embeddingResponse := <-embeddingChannel
	if len(embeddingResponse.Data) > 0 {
		updateChannel := make(chan int)
		defer close(updateChannel)
		go UpdateChatSessionTitleVector(sessionId, embeddingResponse.Data[0].Embedding, updateChannel)
		channel <- <-updateChannel
		return
	}
	channel <- 0
}
