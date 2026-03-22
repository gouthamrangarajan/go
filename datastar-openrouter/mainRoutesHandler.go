package main

import (
	"bytes"
	"context"
	"datastar-openrouter/components"
	"datastar-openrouter/models"
	"datastar-openrouter/services"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
)

var uiSidMap = make(map[string]chan models.LongSSEData)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// style := styles.Get("github") // Use the same style as in goldmark
	// formatter := html.New(html.WithClasses(true))
	// buf := bytes.Buffer{}
	// formatter.WriteCSS(&buf, style)
	// fmt.Println(buf.String())
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionId := 0
	sessionIdStr := chi.URLParam(request, "sessionId")
	searchMenuTxt := strings.TrimSpace(request.URL.Query().Get("search_menu"))
	sessionId, err := strconv.Atoi(sessionIdStr)
	if err != nil {
		sessionId = 0
	}
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == sessionId {
			selectedSession = session
			break
		}
	}
	if selectedSession.Id == 0 && sessionId != 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}

	chatConversationChannel := make(chan []models.ChatConversation)
	defer close(chatConversationChannel)
	go services.GetChatConversationsWithoutMessageAndFileData(userId, sessionId, chatConversationChannel)

	aiModelsChannel := make(chan []models.AIModel)
	defer close(aiModelsChannel)
	go services.GetAiModels(aiModelsChannel)

	if searchMenuTxt != "" {
		sessions = SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: searchMenuTxt,
		})
	}

	chatConversations := <-chatConversationChannel
	aiModels := <-aiModelsChannel
	uiSId := uuid.New().String()
	uiSidMap[uiSId] = make(chan models.LongSSEData)
	components.Main(
		models.UIMainModel{
			Messages:         chatConversations,
			Sessions:         sessions,
			AIModels:         aiModels,
			AllowWebSearch:   selectedSession.AllowWebSearch,
			ImageGeneration:  selectedSession.ImageGeneration,
			CurrentSessionId: sessionId,
			MenuSearchTerm:   searchMenuTxt,
			UiSid:            uiSId,
		}).Render(request.Context(), responseWriter)
}
func longSSEHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)

	sse := datastar.NewSSE(responseWriter, request)

	if clientSignal.SessionId != 0 {

		conversationsChannel := make(chan []models.ChatConversation)

		go services.GetChatConversations(userId, clientSignal.SessionId, conversationsChannel)
		conversations := <-conversationsChannel
		close(conversationsChannel)

		if len(conversations) != 0 {
			markdownToHtmlChannel := make(chan string)
			go services.ConvertConversationMarkdownsToHtml(conversations, markdownToHtmlChannel)
			select {
			case <-request.Context().Done():
				break
			default:
				for _, conversation := range conversations {
					if conversation.FileData != "" {
						sse.PatchElementTempl(components.ChatMessageFileData(conversation), datastar.WithUseViewTransitions(false))
					}
					sse.PatchElementTempl(components.ChatMessageModelIdDisplay(conversation), datastar.WithUseViewTransitions(false))
				}
			}

			for element := range markdownToHtmlChannel {
				select {
				case <-request.Context().Done():
					continue
				default:
					sse.PatchElements(element, datastar.WithUseViewTransitions(false))
					sse.ExecuteScript("window.mermaid.run()", datastar.WithExecuteScriptAutoRemove(true))
				}

			}
		}
	}
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		for {
			select {
			case <-request.Context().Done():
				close(uiSidMap[clientSignal.UiSid])
				delete(uiSidMap, clientSignal.UiSid)
				return

			case data := <-uiSidMap[clientSignal.UiSid]:
				switch {
				case data.IsError:
					services.SendErrorMessageToUI(sse, data.Content)
				case data.IsScript:
					sse.ExecuteScript(data.Content, datastar.WithExecuteScriptAutoRemove(true))
				case data.IsSignal:
					sse.PatchSignals([]byte(data.Content))
				case data.IsRemove:
					sse.RemoveElement(data.Selector, datastar.WithUseViewTransitions(data.UseViewTransition))
				default:
					if strings.TrimSpace(data.Selector) == "" {
						sse.PatchElements(data.Content, datastar.WithUseViewTransitions(data.UseViewTransition))
					} else if data.Mode != nil {
						sse.PatchElements(data.Content, datastar.WithSelector(data.Selector), data.Mode, datastar.WithUseViewTransitions(data.UseViewTransition))
					} else {
						sse.PatchElements(data.Content, datastar.WithSelector(data.Selector), datastar.WithUseViewTransitions(data.UseViewTransition))
					}
					continue
				}
			}
		}
	}
}
func newChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	insertChatSessionChannel := make(chan int)
	defer close(insertChatSessionChannel)
	newSession := models.ChatSession{Title: "New Chat"}
	go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
	newSession.Id = <-insertChatSessionChannel
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		if newSession.Id == 0 {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content: `Failed to create new chat session. Please try again later.`,
				IsError: true,
			}

			return
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  `window.location.href=window.location.origin+'/'+` + strconv.Itoa(newSession.Id),
			IsScript: true,
		}
	}
	embeddingChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(embeddingChannel)
	embeddingRequest := models.VoyageEmbeddingRequest{
		Input: []string{newSession.Title},
	}
	go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
	embeddingResponse := <-embeddingChannel
	if len(embeddingResponse.Data) > 0 {
		updateTitleVectorChannel := make(chan int)
		defer close(updateTitleVectorChannel)
		go services.UpdateChatSessionTitleVector(newSession.Id, embeddingResponse.Data[0].Embedding, updateTitleVectorChannel)
		<-updateTitleVectorChannel
	}

}
func deleteSessionHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	if clientSignal.SessionIdToDelete == 0 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == clientSignal.SessionIdToDelete {
			selectedSession = session
			break
		}
	}
	if selectedSession.Id == 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}
	deleteSessionChannel := make(chan int)
	defer close(deleteSessionChannel)
	go services.DeleteChatSession(userId, clientSignal.SessionIdToDelete, deleteSessionChannel)
	if <-deleteSessionChannel == 0 {
		if _, exists := uiSidMap[clientSignal.UiSid]; exists {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content: `Failed to delete chat session. Please try again later.`,
				IsError: true,
			}
		}
		return
	}
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		if clientSignal.SessionIdToDelete == clientSignal.SessionId {
			componentBuffer := new(bytes.Buffer)
			components.Section([]models.ChatConversation{}).Render(context.Background(), componentBuffer)
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:           componentBuffer.String(),
				Selector:          "section",
				UseViewTransition: true,
				Mode:              datastar.WithModeOuter(),
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  `{sessionId:0,webSearch:false}`,
				IsSignal: true,
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  `window.history.replaceState({},'','/')`,
				IsScript: true,
			}
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			IsRemove:          true,
			Selector:          "#menuItem_" + strconv.Itoa(clientSignal.SessionIdToDelete),
			UseViewTransition: true,
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			IsSignal: true,
			Content:  `{showDeleteModal:false}`}
	}
}
func searchSessionHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	userExistsChannel := make(chan bool)
	defer close(userExistsChannel)
	go services.CheckUserExistsInTable(userId, userExistsChannel)
	if !<-userExistsChannel {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	// sse := datastar.NewSSE(responseWriter, request)
	sessions := []models.ChatSession{}
	clientSignal.SearchMenu = strings.TrimSpace(clientSignal.SearchMenu)
	if clientSignal.SearchMenu == "" {
		sessionsChannel := make(chan []models.ChatSession)
		defer close(sessionsChannel)
		go services.GetChatSessions(userId, sessionsChannel)
		sessions = <-sessionsChannel
	} else {
		sessions = SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: clientSignal.SearchMenu})
	}
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		componentBuffer := new(bytes.Buffer)
		components.MenuUl(sessions, clientSignal.SearchMenu).Render(context.Background(), componentBuffer)
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           componentBuffer.String(),
			UseViewTransition: true,
		}
		scriptToExecute := "window.history.replaceState({},'','/"
		if clientSignal.SessionId != 0 {
			scriptToExecute += strconv.Itoa(clientSignal.SessionId)
		}
		if strings.TrimSpace(clientSignal.SearchMenu) != "" {
			scriptToExecute += "?search_menu=" + clientSignal.SearchMenu
		}
		scriptToExecute += "')"
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  scriptToExecute,
			IsScript: true,
		}

	}
}
func fileUploadHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	if len(clientSignal.FileData) != 1 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	fileDataForRegex := "data:" + clientSignal.FileData[0].Mime + ";base64," + clientSignal.FileData[0].Contents
	fileName := clientSignal.FileData[0].Name
	imgMatches := services.ImgRegex.FindStringSubmatch(fileDataForRegex)
	pdfMatches := services.PdfRegex.FindStringSubmatch(fileDataForRegex)

	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		if (clientSignal.FileData[0].Mime == "application/pdf" && len(pdfMatches) != 2) ||
			(clientSignal.FileData[0].Mime != "application/pdf" && len(imgMatches) != 4) {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			fileName = ""
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content: "Invalid file type. Please upload an file with type (JPG, PNG, WEBP, GIF, PDF)",
				IsError: true,
			}
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
		if err != nil || len(decodedBytes) > 1024*1024 {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content: "File too large. Please upload a file smaller than 1 MB.",
				IsError: true,
			}
			fileName = ""
			return
		}
		bytesBuffer := new(bytes.Buffer)
		components.FileAttachmentDisplay(fileName).Render(context.Background(), bytesBuffer)
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  "{fileUploading:false}",
			IsSignal: true,
		}
	}
}
func removeUploadedFileHandler(responseWriter http.ResponseWriter, request *http.Request) {
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	bytesBuffer := new(bytes.Buffer)

	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		components.FileAttachmentDisplay("").Render(context.Background(), bytesBuffer)
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  "{fileData:'',fileUploading:false}",
			IsSignal: true,
		}
	}
}

// ALGO
// Handle unauthorized user - user does not exist in table or session id coming from client is not valid
// Handle bad request - more than 1 file uploaded, invalid file(non pdf and non image), file size > 1 MB
// Handle new session creation if session id from client is 0, failure return to UI with error message
// Insert user message chat conversation, failure return to UI with error message
// Insert model message chat conversation with empty content
// Call OpenRouter with streaming in a goroutine
// Update chat session title if it's the first message in the session
// Update chat session allow web search & image generation if applicable
// Stream response from OpenRouter to UI
// Wait for title update if called, call embedding and update title vector,
// wait for allow web search update if called
// If message is empty/error, return error message to UI and delete the model message chat conversation, return
// Update model message chat conversation with full content after streaming is done if message is not empty

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	clientSignal.Prompt = strings.TrimSpace(clientSignal.Prompt)
	clientSignal.SearchMenu = strings.TrimSpace(clientSignal.SearchMenu)

	userExistsChannel := make(chan bool)
	defer close(userExistsChannel)
	go services.CheckUserExistsInTable(userId, userExistsChannel)
	if !<-userExistsChannel {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel

	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == clientSignal.SessionId {
			selectedSession = session
			break
		}
	}

	if clientSignal.SessionId != 0 && selectedSession.Id == 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}

	if len(clientSignal.FileData) > 1 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	fileData := ""
	fileName := ""
	if len(clientSignal.FileData) == 1 {
		fileData = "data:" + clientSignal.FileData[0].Mime + ";base64," + clientSignal.FileData[0].Contents
		fileName = clientSignal.FileData[0].Name
		imgMatches := services.ImgRegex.FindStringSubmatch(fileData)
		pdfMatches := services.PdfRegex.FindStringSubmatch(fileData)
		if (clientSignal.FileData[0].Mime == "application/pdf" && len(pdfMatches) != 2) ||
			(clientSignal.FileData[0].Mime != "application/pdf" && len(imgMatches) != 4) {
			http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
		if err != nil || len(decodedBytes) > 1024*1024 {
			http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	if clientSignal.Prompt != "" {
		if clientSignal.SessionId == 0 {
			insertChatSessionChannel := make(chan int)
			defer close(insertChatSessionChannel)
			newSession := models.ChatSession{Title: clientSignal.Prompt}
			go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
			newSession.Id = <-insertChatSessionChannel
			clientSignal.SessionId = newSession.Id
			if clientSignal.SessionId == 0 {
				if _, exists := uiSidMap[clientSignal.UiSid]; exists {
					uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
						Content: `Failed to create new chat session. Please try again later.`,
						IsError: true,
					}
				}
				return
			}
			if _, exists := uiSidMap[clientSignal.UiSid]; exists {
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					Content:  `window.history.replaceState({},'','/` + strconv.Itoa(clientSignal.SessionId) + `')`,
					IsScript: true,
				}
				menuItemBuffer := new(bytes.Buffer)
				components.MenuItem(newSession, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					Content:  menuItemBuffer.String(),
					Mode:     datastar.WithModeAppend(),
					Selector: "#menu",
				}
			}
		}

		insertUserConversationChannel := make(chan int)
		defer close(insertUserConversationChannel)
		userMessageChat := models.ChatConversation{Role: "user", Content: clientSignal.Prompt, SessionId: clientSignal.SessionId, FileName: fileName, FileData: fileData}
		go services.InsertChatConversation(userMessageChat, insertUserConversationChannel)
		userMessageChat.Id = <-insertUserConversationChannel

		if userMessageChat.Id == 0 {
			if _, exists := uiSidMap[clientSignal.UiSid]; exists {
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					Content: `Failed to save chat conversation. Please try again later.`,
					IsError: true,
				}
			}
			return
		}
		if _, exists := uiSidMap[clientSignal.UiSid]; exists {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  `document.getElementById('hint')?.remove();`,
				IsScript: true,
			}
			userMessageBuffer := new(bytes.Buffer)
			components.ChatMessage(userMessageChat, true).Render(context.Background(), userMessageBuffer)
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:           userMessageBuffer.String(),
				UseViewTransition: true,
				Mode:              datastar.WithModeAppend(),
				Selector:          "section",
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  `{prompt:"",sessionId:` + strconv.Itoa(clientSignal.SessionId) + `,fileData:''}`,
				IsSignal: true,
			}
			fileAttachmentBuffer := new(bytes.Buffer)
			components.FileAttachmentDisplay("").Render(context.Background(), fileAttachmentBuffer)
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:           fileAttachmentBuffer.String(),
				UseViewTransition: true,
			}

			markdownToHtmlChannel := make(chan string)
			go services.ConvertConversationMarkdownsToHtml([]models.ChatConversation{userMessageChat}, markdownToHtmlChannel)

			if userMessageChat.FileData != "" {
				chatMessageUserFileDataBuffer := new(bytes.Buffer)
				components.ChatMessageFileData(userMessageChat).Render(context.Background(), chatMessageUserFileDataBuffer)
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					Content:           chatMessageUserFileDataBuffer.String(),
					UseViewTransition: true,
				}
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:           <-markdownToHtmlChannel,
				UseViewTransition: true,
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  `window.mermaid.run()`,
				IsScript: true,
			}
		}
		createModelMessageChatCallOpenRouterUpdateSessionMetadataSendDataToUI(clientSignal, userId, selectedSession)
	}
}

// ALGO
// Handle unauthorized user - user does not exist in table or session id coming from client is not valid
// Handle bad request - message id to retry is 0
// Delete all chat conversations after the message id to retry, remove html elements from UI
// Insert model message chat conversation with empty content
// Call OpenRouter with streaming in a goroutine
// Update chat session title if it's the first message in the session
// Update chat session allow web search & image generation if applicable
// Stream response from OpenRouter to UI
// Wait for title update if called, call embedding and update title vector,
// wait for allow web search update if called
// If message is empty/error, return error message to UI and delete the model message chat conversation, return
// Update model message chat conversation with full content after streaming is done if message is not empty
func retryHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	clientSignal.SearchMenu = strings.TrimSpace(clientSignal.SearchMenu)

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	var selectedSession models.ChatSession
	for _, session := range sessions {
		if session.Id == clientSignal.SessionId {
			selectedSession = session
			break
		}
	}
	if selectedSession.Id == 0 {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}
	if clientSignal.MessageIdToRetry == 0 {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	deleteChannel := make(chan []int)
	defer close(deleteChannel)
	go services.DeleteMessageChatConversationForRetry(models.DeleteChatConversationsAfterAId{
		UserId:                         userId,
		SessionId:                      clientSignal.SessionId,
		ConversationIdAfterWhichDelete: clientSignal.MessageIdToRetry,
	}, deleteChannel)
	deletedIds := <-deleteChannel
	// if len(deletedIds) == 0 {
	// 	services.SendErrorMessageToUI(sse, "Failed to retry chat. Please try again later.")
	// 	return
	// }
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		for _, id := range deletedIds {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				IsRemove:          true,
				Selector:          "#message-" + strconv.Itoa(id),
				UseViewTransition: true,
			}
		}
	}
	createModelMessageChatCallOpenRouterUpdateSessionMetadataSendDataToUI(clientSignal, userId, selectedSession)
}

func createModelMessageChatCallOpenRouterUpdateSessionMetadataSendDataToUI(clientSignal models.ClientSignals, userId string, selectedSession models.ChatSession) {
	insertModelConversationChannel := make(chan int)
	defer close(insertModelConversationChannel)
	modelMessageChat := models.ChatConversation{Role: "assistant", Content: "", SessionId: clientSignal.SessionId, FileData: ""}
	if clientSignal.ImageGeneration {
		modelMessageChat.FileName = "generating_image.png"
	}
	go services.InsertChatConversation(modelMessageChat, insertModelConversationChannel)
	modelMessageChat.Id = <-insertModelConversationChannel
	modelMessageChatBuffer := new(bytes.Buffer)
	components.ChatMessage(modelMessageChat, true).Render(context.Background(), modelMessageChatBuffer)
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           modelMessageChatBuffer.String(),
			UseViewTransition: true,
			Mode:              datastar.WithModeAppend(),
			Selector:          "section",
		}

		if modelMessageChat.Id == 0 {
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:  "Error storing chat conversation.",
				IsScript: true,
			}
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  `document.querySelector("main").scrollTo(0, document.querySelector("main").scrollHeight);`,
			IsScript: true,
		}
	}
	openRouterChannel := make(chan models.OpenRouterModelIdAndDeltaString)
	openRouterRequest, _ := services.GenerateOpenRouterRequest(userId, clientSignal)

	go services.CallOpenRouter(openRouterRequest, openRouterChannel)

	updateTitleChannel := make(chan int)
	defer close(updateTitleChannel)
	embeddingChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(embeddingChannel)
	updateTitleCalled := false
	titleToUpdate := clientSignal.Prompt

	if len(openRouterRequest.Messages) == 1 {
		if strings.TrimSpace(selectedSession.Title) != "New Chat" && strings.TrimSpace(selectedSession.Title) != "" {
			titleToUpdate = selectedSession.Title
		}
		titleToVectorize := titleToUpdate
		if len(titleToVectorize) > 500 {
			titleToVectorize = titleToUpdate[:500]
		}
		go services.UpdateChatSessionTitle(userId, models.ChatSession{Id: clientSignal.SessionId, Title: titleToUpdate}, updateTitleChannel)
		updateTitleCalled = true

		embeddingRequest := models.VoyageEmbeddingRequest{
			Input: []string{titleToVectorize},
		}
		go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
	}

	updateWebSearchChannel := make(chan int)
	defer close(updateWebSearchChannel)
	updateWebSearchCalled := false

	updateImageGenerationChannel := make(chan int)
	defer close(updateImageGenerationChannel)
	updateImageGeneratioCalled := false

	if selectedSession.AllowWebSearch != clientSignal.WebSearch {
		go services.UpdateChatSessionAllowWebSearch(userId, clientSignal.SessionId, clientSignal.WebSearch, updateWebSearchChannel)
		updateWebSearchCalled = true
	}
	if selectedSession.ImageGeneration != clientSignal.ImageGeneration {
		go services.UpdateChatSessionImageGeneration(userId, clientSignal.SessionId, clientSignal.ImageGeneration, updateImageGenerationChannel)
		updateImageGeneratioCalled = true
	}

	for msg := range openRouterChannel {
		if msg.DeltaContent == "Error" {
			fmt.Printf("Error in getting response from OpenRouter\n")
			// handle error
			continue
		}
		modelMessageChat.Content += msg.DeltaContent
		modelMessageChat.FileData += msg.DeltaImage
		modelMessageChat.ModelId = msg.ModelId
		if _, exists := uiSidMap[clientSignal.UiSid]; !exists {
			continue
		}
		// sse.PatchElementTempl(components.ChatMessage(modelMessageChat, true))
		markdownToHtmlChannel := make(chan string)
		go services.ConvertConversationMarkdownsToHtml([]models.ChatConversation{modelMessageChat}, markdownToHtmlChannel)
		if modelMessageChat.FileData != "" {
			modelMessageChat.FileName = "generated_image.png"
			modelMessageChatFileDataBuffer := new(bytes.Buffer)
			components.ChatMessageFileData(modelMessageChat).Render(context.Background(), modelMessageChatFileDataBuffer)
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content:           modelMessageChatFileDataBuffer.String(),
				UseViewTransition: true,
			}
		} else {
			modelMessageChat.FileName = ""
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           <-markdownToHtmlChannel,
			UseViewTransition: true,
		}
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:  "window.mermaid.run()",
			IsScript: true,
		}
	}
	if _, exists := uiSidMap[clientSignal.UiSid]; exists {
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			IsRemove: true,
			Selector: "#thinkingMesssage",
		}
		chatMessageModelIdBuffer := new(bytes.Buffer)
		components.ChatMessageModelIdDisplay(modelMessageChat).Render(context.Background(), chatMessageModelIdBuffer)
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content:           chatMessageModelIdBuffer.String(),
			UseViewTransition: true,
		}
	}

	if updateTitleCalled {
		if <-updateTitleChannel != 0 {
			if _, exists := uiSidMap[clientSignal.UiSid]; exists {
				menuItemBuffer := new(bytes.Buffer)
				components.MenuItem(models.ChatSession{Id: clientSignal.SessionId, Title: titleToUpdate}, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					Content: menuItemBuffer.String(),
				}
			}
		}
		embeddingResponse := <-embeddingChannel
		if len(embeddingResponse.Data) > 0 {
			updateTitleVectorChannel := make(chan int)
			defer close(updateTitleVectorChannel)
			go services.UpdateChatSessionTitleVector(clientSignal.SessionId, embeddingResponse.Data[0].Embedding, updateTitleVectorChannel)
			<-updateTitleVectorChannel
		}
	}

	if updateWebSearchCalled {
		<-updateWebSearchChannel
	}
	if updateImageGeneratioCalled {
		<-updateImageGenerationChannel
	}
	if (modelMessageChat.Content == "" || strings.TrimSpace(modelMessageChat.Content) == "Error") &&
		modelMessageChat.FileData == "" {
		deleteModelConversationChannel := make(chan int)
		defer close(deleteModelConversationChannel)
		go services.DeleteMessageChatConversation(modelMessageChat.Id, deleteModelConversationChannel)
		if _, exists := uiSidMap[clientSignal.UiSid]; exists {
			if <-deleteModelConversationChannel != 0 {
				uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
					IsRemove: true,
					Selector: "#message-" + strconv.Itoa(modelMessageChat.Id),
				}
			}
			uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
				Content: "Unable to get response from AI. Please try again, or switch to a different model.",
				IsError: true,
			}
		}
	}

	updateModelConversationChannel := make(chan int)
	defer close(updateModelConversationChannel)
	go services.UpateMessageChatConversation(models.UpdateChatConversation{
		Id:       modelMessageChat.Id,
		Content:  modelMessageChat.Content,
		ModelId:  modelMessageChat.ModelId,
		FileData: modelMessageChat.FileData,
		FileName: modelMessageChat.FileName,
	}, updateModelConversationChannel)
	rowsAffected := <-updateModelConversationChannel
	if _, exists := uiSidMap[clientSignal.UiSid]; exists && rowsAffected == 0 {
		uiSidMap[clientSignal.UiSid] <- models.LongSSEData{
			Content: "Failed to update chat conversation. Please try again later.",
			IsError: true,
		}
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
	go services.CallVoyageEmbedding(embeddingRequest, embeddingsChannel)
	embeddingResponse := <-embeddingsChannel
	if len(embeddingResponse.Data) > 0 {
		go services.SearchChatSessions(data.UserId, embeddingResponse.Data[0].Embedding, searchSessionsChannel)
		retVal = <-searchSessionsChannel
	}
	return retVal
}
