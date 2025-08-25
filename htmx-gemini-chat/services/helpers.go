package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"htmx-gemini-chat/models"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userId"

var imgRegex = regexp.MustCompile(`^data:(image/(png|jpeg|jpg|webp));base64,([A-Za-z0-9+/=]+)$`)
var pdfRegex = regexp.MustCompile(`^data:application/pdf;base64,([A-Za-z0-9+/=]+)$`)

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
func InsertChatSessionViaChannel(userId string, title string, allowWebSearch bool) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	defer close(insertSessionChannel)
	go InsertChatSession(userId, title, allowWebSearch, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}

func GenerateGeminiRequest(userId string, sessionId int, prompt string, fileBase64 string, allowWebSearch bool) (models.GeminiRequest, string) {
	err := ""
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, sessionId, conversationsChannel)
	conversations := <-conversationsChannel
	geminiRequest := models.GeminiRequest{}
	geminiRequest.Contents = make([]models.GeminiRequestContent, 0, len(conversations)+1)
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.Message) != "" {
			if conversation.FileData != "" {
				messageToGeminiRequestContent := models.GeminiRequestContent{
					Role: conversation.Sender,
					Parts: append(make([]models.GeminiRequestParts, 0, 2), models.GeminiRequestParts{
						Text: &conversation.Message,
					}),
				}
				imgMatches := imgRegex.FindStringSubmatch(conversation.FileData)
				pdfMatches := pdfRegex.FindStringSubmatch(conversation.FileData)
				if len(imgMatches) == 4 {
					messageToGeminiRequestContent.Parts = append(messageToGeminiRequestContent.Parts, models.GeminiRequestParts{
						FileData: &models.GeminiRequestFileData{
							MimeType: imgMatches[1],
							Data:     imgMatches[3],
						},
					})
				} else if len(pdfMatches) == 2 {
					messageToGeminiRequestContent.Parts = append(messageToGeminiRequestContent.Parts, models.GeminiRequestParts{
						FileData: &models.GeminiRequestFileData{
							MimeType: "application/pdf",
							Data:     pdfMatches[1],
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
	if fileBase64 != "" {
		partsCapacityForPrompt = 2
	}
	promptToGeminiRequestContent := models.GeminiRequestContent{
		Role: "user",
		Parts: append(make([]models.GeminiRequestParts, 0, partsCapacityForPrompt), models.GeminiRequestParts{
			Text: &prompt,
		}),
	}
	if fileBase64 != "" {
		strDataMatch := ""
		mimeType := ""
		imgMatches := imgRegex.FindStringSubmatch(fileBase64)
		pdfMatches := pdfRegex.FindStringSubmatch(fileBase64)
		if len(imgMatches) != 4 && len(pdfMatches) != 2 {
			err = "Invalid data"
		} else {

			if len(imgMatches) == 4 {
				strDataMatch = imgMatches[3]
				mimeType = imgMatches[1]
			} else {
				strDataMatch = pdfMatches[1]
				mimeType = "application/pdf"
			}
			decoded, decodeErr := base64.StdEncoding.DecodeString(strDataMatch)
			if decodeErr != nil || len(decoded) > 1024*1024 {
				err = "Invalid File Size"
			}
		}

		if err == "" {
			promptToGeminiRequestContent.Parts = append(promptToGeminiRequestContent.Parts, models.GeminiRequestParts{
				FileData: &models.GeminiRequestFileData{
					MimeType: mimeType,
					Data:     strDataMatch,
				},
			})
		}
	}
	geminiRequest.Contents = append(geminiRequest.Contents, promptToGeminiRequestContent)
	if allowWebSearch {
		geminiRequest.Tools = make(map[string]interface{})
		geminiRequest.Tools["google_search"] = struct{}{}
	}
	return geminiRequest, err
}

func GenerateGeminiEmbeddingRequest(srchTxt string) models.GeminiEmbeddingRequest {
	geminiRequestParts := []models.GeminiRequestParts{}
	geminiRequestParts = append(geminiRequestParts, models.GeminiRequestParts{
		Text: &srchTxt,
	})

	return models.GeminiEmbeddingRequest{
		// Config: models.GeminiEmbeddingRequestConfig{OutputDimension: 768},
		Content: models.GeminiRequestContent{
			Role:  "user",
			Parts: geminiRequestParts,
		}}
}
