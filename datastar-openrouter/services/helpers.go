package services

import (
	"bytes"
	"datastar-openrouter/models"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"go.abhg.dev/goldmark/mermaid"
)

type contextKey string

const UserIDKey contextKey = "userId"

var ImgRegex = regexp.MustCompile(`^data:(image/(png|jpeg|jpg|webp|gif));base64,([A-Za-z0-9+/=]+)$`)
var PdfRegex = regexp.MustCompile(`^data:application/pdf;base64,([A-Za-z0-9+/=]+)$`)

const copySvg = `<svg
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="currentColor"
					class="size-5 pointer-events-none"
				>
					<path d="M7.5 3.375c0-1.036.84-1.875 1.875-1.875h.375a3.75 3.75 0 0 1 3.75 3.75v1.875C13.5 8.161 14.34 9 15.375 9h1.875A3.75 3.75 0 0 1 21 12.75v3.375C21 17.16 20.16 18 19.125 18h-9.75A1.875 1.875 0 0 1 7.5 16.125V3.375Z"></path>
					<path d="M15 5.25a5.23 5.23 0 0 0-1.279-3.434 9.768 9.768 0 0 1 6.963 6.963A5.23 5.23 0 0 0 17.25 7.5h-1.875A.375.375 0 0 1 15 7.125V5.25ZM4.875 6H6v10.125A3.375 3.375 0 0 0 9.375 19.5H16.5v1.125c0 1.035-.84 1.875-1.875 1.875h-9.75A1.875 1.875 0 0 1 3 20.625V7.875C3 6.839 3.84 6 4.875 6Z"></path>
				</svg>`
const systemPrompt = `You are Nexus AI, a highly advanced unified AI interface. 
						Your goal is to provide accurate, context-aware, and helpful responses by utilizing your multi-modal capabilities (analyzing images, PDFs, and text) and your advanced reasoning.

						### GUIDELINES:
						1. IDENTITY: You are Nexus AI. Do not identify as a specific model (e.g., GPT-4, Claude, or Gemini) unless explicitly asked about your underlying architecture. 
						2. TONE: Professional, concise, and helpful. Avoid "fluff" or overly robotic standard openings (e.g., skip "As an AI language model...").
						3. CAPABILITIES:
						- You can analyze uploaded documents (PDFs) and images provided by the user.
						- You can generate code across various languages (GO, Python, JS, etc.).						
						4. FORMATTING:
						- Use Markdown for all formatting. 
						- Use triple backticks for code blocks and always specify the language.
						- Use LaTeX for mathematical formulas.
						- If a response is long, use headers and bullet points for readability.
						5. CONTEXT: Always consider the previous chat history.

						Current Date: %v`

func GenerateUserSessionKey(userId string, sessionId string) string {
	return fmt.Sprintf("%s-%s", userId, sessionId)
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
	openRouterRequest.Messages = make([]models.OpenRouterRequestMessage, 0, len(conversations)+2)
	openRouterRequest.Messages = append(openRouterRequest.Messages, models.OpenRouterRequestMessage{
		Role:    "system",
		Content: fmt.Sprintf(systemPrompt, time.Now().Format("January 2, 2006")),
	})
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
		// 	if conversation.Role == "assistant" {
		// 		conversation.Content = "```" + `mermaid
		// 	graph TD
		// subgraph Client_Side [User Access]
		//     User((User))
		// end

		// subgraph Edge_Location [Content Delivery]
		//     CF[AWS CloudFront]
		//     S3[(AWS S3 Assets)]
		// end

		// subgraph Public_Subnet [Entry Point]
		//     AGW[AWS API Gateway]
		// end

		// subgraph Private_Subnet [Compute & Data]
		//     ALB[AWS Application Load Balancer]
		//     Lambda[AWS Lambda]
		//     DocDB[(AWS DocumentDB)]
		// end

		// %% Flow Connections
		// User -->|Requests Content/API| CF
		// CF -->|Fetch Static Assets| S3
		// CF -->|Forward API Calls| AGW
		// AGW --> ALB
		// ALB --> Lambda
		// Lambda -->|Query/Write| DocDB

		// %% Styling
		// style CF fill:#FF9900,stroke:#232F3E,color:white
		// style S3 fill:#3F8624,stroke:#232F3E,color:white
		// style AGW fill:#8C3123,stroke:#232F3E,color:white
		// style ALB fill:#8C3123,stroke:#232F3E,color:white
		// style Lambda fill:#FF9900,stroke:#232F3E,color:white
		// style DocDB fill:#3156CF,stroke:#232F3E,color:white
		// 	` + "\n```"
		// 	}
		var buf bytes.Buffer
		md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.DefinitionList,
			extension.Footnote, extension.Typographer, extension.CJK, &mermaid.Extender{
				RenderMode:   mermaid.RenderModeClient,
				ContainerTag: "div",
				NoScript:     true,
			},
			highlighting.NewHighlighting(highlighting.WithStyle("dracula"))))
		if err := md.Convert([]byte(conversation.Content), &buf); err != nil {
			fmt.Printf("Error converting markdown: %v\n", err)
			channel <- ""
			return
		}
		mkdwn := buf.String()
		// fmt.Printf("mkdwn,%v\n", mkdwn)
		preRegex := regexp.MustCompile(`<pre`)
		mkdwn = preRegex.ReplaceAllString(mkdwn, `<div class="relative"><pre`)
		preEndRegex := regexp.MustCompile(`</pre>`)
		mkdwn = preEndRegex.ReplaceAllString(mkdwn, `<button class="appearance-none outline-none absolute top-1.5 right-1.5 text-white p-1 rounded-full w-8 h-8 flex items-center justify-center cursor-pointer hover:ring-1 hover:ring-white focus:ring-1 focus:ring-white"
														alt="Copy to Clipboard"
														data-on:click__viewtransition="window.navigator.clipboard.writeText((evt.srcElement.previousElementSibling || evt.srcElement.parentElement).innerText.replaceAll('\n\n','\n')).then(()=>{evt.srcElement.innerHTML=document.getElementById('copiedSvg').innerHTML; setTimeout(()=>{evt.srcElement.innerHTML=document.getElementById('copySvg').innerHTML},2000)})">
													`+copySvg+`
													</button>
													</pre></div>`)
		channel <- "<div id='markdown_" + strconv.Itoa(conversation.Id) + "' class='prose dark:prose-invert'>" + mkdwn + "</div>"
	}
}

func SearchSessionsViaChannel(data models.SearchSessionViaChannelRequest) []models.ChatSession {
	retVal := []models.ChatSession{}
	searchSessionsChannel := make(chan []models.ChatSession)
	defer close(searchSessionsChannel)
	embeddingsChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(embeddingsChannel)

	embeddingRequest := models.VoyageEmbeddingRequest{
		Input: []string{data.SearchTerm},
	}
	go CallVoyageEmbedding(embeddingRequest, embeddingsChannel)
	embeddingResponse := <-embeddingsChannel
	if len(embeddingResponse.Data) > 0 {
		go SearchChatSessions(data.UserId, embeddingResponse.Data[0].Embedding, searchSessionsChannel)
		retVal = <-searchSessionsChannel
	}
	return retVal
}
