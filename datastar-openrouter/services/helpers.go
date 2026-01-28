package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"datastar-openrouter/models"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ImgRegex = regexp.MustCompile(`^data:(image/(png|jpeg|jpg|webp|gif));base64,([A-Za-z0-9+/=]+)$`)
var PdfRegex = regexp.MustCompile(`^data:application/pdf;base64,([A-Za-z0-9+/=]+)$`)

const copySvg = `<svg
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="currentColor"
					class="size-6 pointer-events-none"
				>
					<path d="M7.5 3.375c0-1.036.84-1.875 1.875-1.875h.375a3.75 3.75 0 0 1 3.75 3.75v1.875C13.5 8.161 14.34 9 15.375 9h1.875A3.75 3.75 0 0 1 21 12.75v3.375C21 17.16 20.16 18 19.125 18h-9.75A1.875 1.875 0 0 1 7.5 16.125V3.375Z"></path>
					<path d="M15 5.25a5.23 5.23 0 0 0-1.279-3.434 9.768 9.768 0 0 1 6.963 6.963A5.23 5.23 0 0 0 17.25 7.5h-1.875A.375.375 0 0 1 15 7.125V5.25ZM4.875 6H6v10.125A3.375 3.375 0 0 0 9.375 19.5H16.5v1.125c0 1.035-.84 1.875-1.875 1.875h-9.75A1.875 1.875 0 0 1 3 20.625V7.875C3 6.839 3.84 6 4.875 6Z"></path>
				</svg>`

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
	}

	clientSignal.ModelId += ":nitro"

	openRouterRequest := models.OpenRouterRequest{
		Stream: true,
		Model:  clientSignal.ModelId,
	}
	openRouterRequest.Modalities = []string{"text"}
	if clientSignal.ImageGeneration {
		openRouterRequest.Modalities = []string{"text", "image"}
		openRouterRequest.Stream = false
	}
	if clientSignal.WebSearch {
		openRouterRequest.Plugins = []map[string]string{
			{
				"id": "web",
			},
		}
		openRouterRequest.Stream = false
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
func ConvertConversationMarkdownsToHtml(conversations []models.ChatConversation, channel chan<- string) {
	defer close(channel)
	for _, conversation := range conversations {
		var buf bytes.Buffer
		md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList,
			extension.Footnote, extension.Typographer, extension.CJK,
			highlighting.NewHighlighting(highlighting.WithStyle("dracula"))))
		if err := md.Convert([]byte(conversation.Content), &buf); err != nil {
			fmt.Printf("Error converting markdown: %v\n", err)
			channel <- ""
			return
		}
		mkdwn := buf.String()
		preRegex := regexp.MustCompile(`<pre`)
		mkdwn = preRegex.ReplaceAllString(mkdwn, `<pre class="relative"`)
		preEndRegex := regexp.MustCompile(`</pre>`)
		mkdwn = preEndRegex.ReplaceAllString(mkdwn, `<button class="appearance-none outline-none absolute top-2 right-2 text-white p-1 rounded-full w-8 h-8 flex items-center justify-center cursor-pointer hover:ring-1 hover:ring-white focus:ring-1 focus:ring-white"
														alt="Copy to Clipboard"
														data-on:click__viewtransition="window.navigator.clipboard.writeText(evt.srcElement.previousElementSibling.innerText.replaceAll('\n\n','\n')).then(()=>{evt.srcElement.innerHTML=document.getElementById('copiedSvg').innerHTML; setTimeout(()=>{evt.srcElement.innerHTML=document.getElementById('copySvg').innerHTML},2000)})">
													`+copySvg+`
													</button>
													</pre>`)
		channel <- "<div id='markdown_" + strconv.Itoa(conversation.Id) + "' class='prose'>" + mkdwn + "</div>"
	}
}
