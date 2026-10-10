package services

import (
	"bytes"
	"crypto/rand"
	"datastar-openrouter/internal/models"
	openrouter "datastar-openrouter/services/open-router"
	"datastar-openrouter/services/voyage"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"go.abhg.dev/goldmark/mermaid"
)

type HelperService struct {
	imgRegex       *regexp.Regexp
	pdfRegex       *regexp.Regexp
	copySvg        string
	systemPrompt   string
	dbService      *DBService
	voyageClient   *voyage.Client
	defaultModelId string
}
type contextKey string

func NewHelperService(dbService *DBService, voyageClient *voyage.Client) *HelperService {
	imgRegex, err := regexp.Compile(os.Getenv("IMG_REGEX"))
	if err != nil {
		fmt.Printf("Error compiling IMG_REGEX: %v\n", err)
	}
	pdfRegex, err := regexp.Compile(os.Getenv("PDF_REGEX"))
	if err != nil {
		fmt.Printf("Error compiling PDF_REGEX: %v\n", err)
	}
	return &HelperService{
		defaultModelId: os.Getenv("DEFAULT_MODEL_ID"),
		imgRegex:       imgRegex,
		pdfRegex:       pdfRegex,
		copySvg: `<svg
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="currentColor"
					class="size-5 pointer-events-none"
				>
					<path d="M7.5 3.375c0-1.036.84-1.875 1.875-1.875h.375a3.75 3.75 0 0 1 3.75 3.75v1.875C13.5 8.161 14.34 9 15.375 9h1.875A3.75 3.75 0 0 1 21 12.75v3.375C21 17.16 20.16 18 19.125 18h-9.75A1.875 1.875 0 0 1 7.5 16.125V3.375Z"></path>
					<path d="M15 5.25a5.23 5.23 0 0 0-1.279-3.434 9.768 9.768 0 0 1 6.963 6.963A5.23 5.23 0 0 0 17.25 7.5h-1.875A.375.375 0 0 1 15 7.125V5.25ZM4.875 6H6v10.125A3.375 3.375 0 0 0 9.375 19.5H16.5v1.125c0 1.035-.84 1.875-1.875 1.875h-9.75A1.875 1.875 0 0 1 3 20.625V7.875C3 6.839 3.84 6 4.875 6Z"></path>
				</svg>`,
		systemPrompt: `You are Nexus AI, a highly advanced unified AI interface.
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

						Current Date: %v`,
		dbService:    dbService,
		voyageClient: voyageClient,
	}
}

func (h *HelperService) GenerateUserSessionKey(userId string, sessionId string) string {
	return fmt.Sprintf("%s-%s", userId, sessionId)
}
func (h *HelperService) GetChatSessionsViaChannel(userId string) []models.ChatSession {
	sessionChannel := make(chan []models.ChatSession)
	go h.dbService.GetChatSessions(userId, sessionChannel)
	sessions := <-sessionChannel
	return sessions
}
func (h *HelperService) InsertChatSessionViaChannel(userId string, data models.ChatSession) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	go h.dbService.InsertChatSession(userId, data, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}

func (h *HelperService) GenerateOpenRouterCompletionsRequest(userId string, clientSignal models.ClientSignals) (openrouter.CompletionsRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	go h.dbService.GetChatConversations(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel
	if strings.TrimSpace(clientSignal.ModelId) == "" {
		clientSignal.ModelId = h.defaultModelId
	}

	clientSignal.ModelId += ":nitro"

	openRouterRequest := openrouter.CompletionsRequest{
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
	openRouterRequest.Messages = make([]openrouter.RequestMessage, 0, len(conversations)+2)
	openRouterRequest.Messages = append(openRouterRequest.Messages, openrouter.RequestMessage{
		Role:    "system",
		Content: fmt.Sprintf(h.systemPrompt, time.Now().Format("January 2, 2006")),
	})
	for _, conversation := range conversations {
		if strings.TrimSpace(conversation.FileData) != "" {
			messageToAppend := openrouter.RequestMessage{
				Role: conversation.Role,
			}
			messageToAppend.ContentWithFileData = append(messageToAppend.ContentWithFileData,
				openrouter.RequestMessageContentWithFileData{
					Type: "text",
					Text: conversation.Content,
				})
			var contentWithFileData openrouter.RequestMessageContentWithFileData
			if h.imgRegex.MatchString(conversation.FileData) {
				contentWithFileData = openrouter.RequestMessageContentWithFileData{
					Type: "image_url",
					ImageUrl: struct {
						Url string `json:"url,omitempty"`
					}{Url: conversation.FileData},
				}
			} else if h.pdfRegex.MatchString(conversation.FileData) {
				contentWithFileData = openrouter.RequestMessageContentWithFileData{
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
			openRouterRequest.Messages = append(openRouterRequest.Messages, openrouter.RequestMessage{
				Role:    conversation.Role,
				Content: conversation.Content,
			})

		}
	}

	// fmt.Printf("Generated OpenRouter Request: %+v\n", openRouterRequest)
	return openRouterRequest, errToRet
}
func (h *HelperService) SendErrorMessageToUI(sse *datastar.ServerSentEventGenerator, message string) {
	sse.PatchSignals([]byte(`{showErrorMessage:true,errorMessage:'` + message + `'}`))
	time.Sleep(3000 * time.Millisecond)
	sse.PatchSignals([]byte("{showErrorMessage:false}"))
}

func (h *HelperService) SearchSessionsViaChannel(data models.SearchSessionViaChannelRequest) []models.ChatSession {
	retVal := []models.ChatSession{}
	searchSessionsChannel := make(chan []models.ChatSession)
	embeddingsChannel := make(chan voyage.Response)

	embeddingRequest := voyage.Request{
		Input: []string{data.SearchTerm},
	}
	go h.voyageClient.CreateEmbedding(embeddingRequest, embeddingsChannel)
	embeddingResponse := <-embeddingsChannel
	if len(embeddingResponse.Data) > 0 {
		go h.dbService.SearchChatSessions(data.UserId, embeddingResponse.Data[0].Embedding, searchSessionsChannel)
		retVal = <-searchSessionsChannel
	}
	return retVal
}
func (h *HelperService) ConvertConversationMarkdownToHtmlAndSendToUserSessionChannel(uiSidMap *sync.Map, clientSignal models.ClientSignals, userId string) {
	userSessionKey := h.GenerateUserSessionKey(userId, clientSignal.UiSid)
	conversationsChannel := make(chan []models.ChatConversation)

	go h.dbService.GetChatConversationsWithoutFileData(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel

	if len(conversations) != 0 {
		markdownToHtmlChannel := make(chan models.ChatConversationMarkdownToHtml)
		go h.ConvertConversationMarkdownsToHtml(conversations, markdownToHtmlChannel)

		for element := range markdownToHtmlChannel {
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: element.Html,
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  `window.mermaid.run()`,
					IsScript: true,
				}
			}
		}
	}
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `{pageLoading:false}`,
			IsSignal: true,
		}
	}
}
func (h *HelperService) ConvertConversationMarkdownsToHtml(conversations []models.ChatConversation, channel chan<- models.ChatConversationMarkdownToHtml) {
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
			fmt.Printf("Error converting markdown to html: %v\n", err)
			channel <- models.ChatConversationMarkdownToHtml{Html: "", ConversationId: conversation.Id}
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
													`+h.copySvg+`
													</button>
													</pre></div>`)
		channel <- models.ChatConversationMarkdownToHtml{Html: "<div id='markdownToHtml_" + strconv.Itoa(conversation.Id) + "' class='prose dark:prose-invert'>" + mkdwn + "</div>", ConversationId: conversation.Id}
	}
}

// randomBytes fills n bytes with cryptographically secure randomness.
func (h *HelperService) randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("rand read: %w", err)
	}
	return b, nil
}

// generateManualCode returns an N-digit code with no leading bias.
// It rejects values above the largest multiple of 10^N that fits in uint32
// to avoid modulo bias.
func (h *HelperService) GenerateManualCode(digits int) (string, error) {
	max := uint32(1)
	for i := 0; i < digits; i++ {
		max *= 10
	}
	// Largest multiple of max that fits in uint32 range.
	limit := (^uint32(0) / max) * max

	for {
		b, err := h.randomBytes(4)
		if err != nil {
			return "", err
		}
		n := binary.BigEndian.Uint32(b)
		if n >= limit {
			continue // reject to avoid bias
		}
		return fmt.Sprintf("%0*d", digits, n%max), nil
	}
}
func (h *HelperService) GenerateOpenRouterImageGenerationRequest(userId string, clientSignal models.ClientSignals) (openrouter.ImageGenerationRequest, string) {
	errToRet := ""
	conversationsChannel := make(chan []models.ChatConversation)
	go h.dbService.GetChatConversations(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel

	openRouterRequest := openrouter.ImageGenerationRequest{
		Prompt: clientSignal.Prompt,
		Stream: false,
		Model:  clientSignal.ModelId,
	}
	// fmt.Printf("Enter here conversations %v\n ", conversations)
	if len(conversations) <= 2 && strings.TrimSpace(clientSignal.Prompt) == "" { //Retry
		openRouterRequest.Prompt = conversations[0].Content
	} else {
		for _, conversation := range conversations {
			if strings.TrimSpace(conversation.FileData) != "" {
				openRouterRequest.InputReference = append(openRouterRequest.InputReference, struct {
					Type     string `json:"type"`
					ImageUrl struct {
						Url string `json:"url"`
					} `json:"image_url"`
				}{
					Type: "image_url",
					ImageUrl: struct {
						Url string `json:"url"`
					}{Url: conversation.FileData},
				})
			}
		}
	}

	// fmt.Printf("Generated OpenRouter Request: %+v\n", openRouterRequest)
	return openRouterRequest, errToRet
}
