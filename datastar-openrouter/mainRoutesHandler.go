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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

var uiSidMap sync.Map

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

	chatConversations := <-chatConversationChannel

	if request.Header.Get("Datastar-Request") == "true" {
		sessionChangeHandler(request, models.SessionChangeData{
			UserId:            userId,
			Session:           selectedSession,
			ChatConversations: chatConversations,
			SearchMenuText:    searchMenuTxt,
		})
		return
	}

	aiModelsChannel := make(chan []models.AIModel)
	defer close(aiModelsChannel)
	go services.GetAiModels(aiModelsChannel)

	if searchMenuTxt != "" {
		sessions = services.SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: searchMenuTxt,
		})
	}
	aiModels := <-aiModelsChannel

	components.Main(
		models.UIMainModel{
			Messages:         chatConversations,
			Sessions:         sessions,
			AIModels:         aiModels,
			AllowWebSearch:   selectedSession.AllowWebSearch,
			ImageGeneration:  selectedSession.ImageGeneration,
			CurrentSessionId: sessionId,
			MenuSearchTerm:   searchMenuTxt,
		}).Render(request.Context(), responseWriter)
}

func longSSEHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	// fmt.Printf("uisid from client %v\n", clientSignal.UiSid)

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)

	userSessionChannel := make(chan models.LongSSEData, 16)
	uiSidMap.Store(userSessionKey, userSessionChannel)

	if clientSignal.SessionId != 0 {
		go sendConversationsMarkdown(clientSignal, userId)
	}

	sse := datastar.NewSSE(responseWriter, request)

	heartBeatTicker := time.NewTicker(5 * time.Second)
	defer heartBeatTicker.Stop()

	sse.PatchSignals([]byte(`{showErrorMessage:false}`))
	for {
		select {
		case <-request.Context().Done():
			uiSidMap.Delete(userSessionKey)
			return
		case data := <-userSessionChannel:
			if channelInMap, ok := uiSidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
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
			}
		case <-heartBeatTicker.C:
			if channelInMap, ok := uiSidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			sse.PatchElementTempl(components.LiveIndicator(), datastar.WithUseViewTransitions(false))
		}
	}
}
func sendConversationsMarkdown(clientSignal models.ClientSignals, userId string) {
	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	conversationsChannel := make(chan []models.ChatConversation)

	go services.GetChatConversationsWithoutFileData(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel
	close(conversationsChannel)

	if len(conversations) != 0 {
		markdownToHtmlChannel := make(chan string)
		go services.ConvertConversationMarkdownsToHtml(conversations, markdownToHtmlChannel)

		for _, conversation := range conversations {
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				if conversation.FileData != "" {
					fileDataDataBuffer := new(bytes.Buffer)
					components.ChatMessageFileData(conversation, true).Render(context.Background(), fileDataDataBuffer)
					userSession.(chan models.LongSSEData) <- models.LongSSEData{
						Content: fileDataDataBuffer.String(),
					}
				}
				modelIdDisplayDataBuffer := new(bytes.Buffer)
				components.ChatMessageModelIdDisplay(conversation).Render(context.Background(), modelIdDisplayDataBuffer)
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: modelIdDisplayDataBuffer.String(),
				}
			}
		}

		for element := range markdownToHtmlChannel {
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: element,
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  `window.mermaid.run()`,
					IsScript: true,
				}
			}
		}
	}
}

func getImageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)

	fileDataChannel := make(chan models.ChatConversation)
	defer close(fileDataChannel)

	go services.GetChatConversationFileData(models.GetConversationRequest{SessionId: clientSignal.SessionId,
		ConversationId: clientSignal.MessageIdToFetchImage, UserId: userId}, fileDataChannel)

	converstationWithFileData := <-fileDataChannel
	if converstationWithFileData.FileData != "" {
		imageDataBuffer := new(bytes.Buffer)
		components.ChatMessageImageDisplayOnHover(converstationWithFileData).Render(context.Background(), imageDataBuffer)
		if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: imageDataBuffer.String(),
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				IsSignal: true,
				Content: `{showImage_` + strconv.Itoa(clientSignal.MessageIdToFetchImage) + `:true,
							imageFetched_` + strconv.Itoa(clientSignal.MessageIdToFetchImage) + `:true}`,
			}
		}
	}
}
func sessionChangeHandler(request *http.Request, data models.SessionChangeData) {
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	clientSignal.SessionId = data.Session.Id
	userSessionKey := services.GenerateUserSessionKey(data.UserId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		dataBuffer := new(bytes.Buffer)
		components.Section(data.ChatConversations).Render(context.Background(), dataBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           dataBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content: `{sessionId:` + strconv.Itoa(data.Session.Id) + `,webSearch:` + strconv.FormatBool(data.Session.AllowWebSearch) +
				`,imageGeneration:` + strconv.FormatBool(data.Session.ImageGeneration) +
				`,messageIdToFetchImage:0,showMenu:false,showErrorMessage:false,showDeleteModal:false
					,sessionIdToDelete:0}`,
			UseViewTransition: true,
		}
		urlToReplace := `/` + strconv.Itoa(data.Session.Id)
		if strings.TrimSpace(data.SearchMenuText) != "" {
			urlToReplace += "?search_menu=" + data.SearchMenuText
		}
		// fmt.Printf("Replacing URL with: %s\n", urlToReplace)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `window.history.replaceState({},"","` + urlToReplace + `")`,
			IsScript: true,
		}
	}
	go sendConversationsMarkdown(clientSignal, data.UserId)
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

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	userSession, userSessionExists := uiSidMap.Load(userSessionKey)

	if userSessionExists {
		if newSession.Id == 0 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: `Failed to create new chat session. Please try again later.`,
				IsError: true,
			}

			return
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content: `{sessionId:` + strconv.Itoa(newSession.Id) + `,webSearch:false,imageGeneration:false,
				messageIdToFetchImage:0,showMenu:false,showErrorMessage:false,showDeleteModal:false,
				sessionIdToDelete:0}`,
			UseViewTransition: true,
		}
		sectionComponentBuffer := new(bytes.Buffer)
		components.Section([]models.ChatConversation{}).Render(context.Background(), sectionComponentBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           sectionComponentBuffer.String(),
			Selector:          "section",
			UseViewTransition: true,
			Mode:              datastar.WithModeOuter(),
		}
		urlToReplace := `/` + strconv.Itoa(newSession.Id)
		if strings.TrimSpace(clientSignal.SearchMenu) != "" {
			urlToReplace += "?search_menu=" + clientSignal.SearchMenu
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `window.history.replaceState({},"","` + urlToReplace + `")`,
			IsScript: true,
		}
		menuItemBuffer := new(bytes.Buffer)
		components.MenuItem(newSession, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           menuItemBuffer.String(),
			Selector:          "#menu",
			UseViewTransition: true,
			Mode:              datastar.WithModeAppend(),
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

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	userSession, userSessionExists := uiSidMap.Load(userSessionKey)

	if <-deleteSessionChannel == 0 {
		if userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: `Failed to delete chat session. Please try again later.`,
				IsError: true,
			}
		}
		return
	}
	if userSessionExists {
		if clientSignal.SessionIdToDelete == clientSignal.SessionId {
			componentBuffer := new(bytes.Buffer)
			components.Section([]models.ChatConversation{}).Render(context.Background(), componentBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           componentBuffer.String(),
				Selector:          "section",
				UseViewTransition: true,
				Mode:              datastar.WithModeOuter(),
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `{sessionId:0,webSearch:false}`,
				IsSignal: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `window.history.replaceState({},'','/')`,
				IsScript: true,
			}
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsRemove:          true,
			Selector:          "#menuItem_" + strconv.Itoa(clientSignal.SessionIdToDelete),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
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
		sessions = services.SearchSessionsViaChannel(models.SearchSessionViaChannelRequest{
			UserId:     userId,
			SearchTerm: clientSignal.SearchMenu})
	}

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		componentBuffer := new(bytes.Buffer)
		components.MenuUl(sessions, clientSignal.SearchMenu).Render(context.Background(), componentBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
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
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  scriptToExecute,
			IsScript: true,
		}

	}
}
func fileUploadHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
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

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		if (clientSignal.FileData[0].Mime == "application/pdf" && len(pdfMatches) != 2) ||
			(clientSignal.FileData[0].Mime != "application/pdf" && len(imgMatches) != 4) {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			fileName = ""
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "Invalid file type. Please upload an file with type (JPG, PNG, WEBP, GIF, PDF)",
				IsError: true,
			}
			return
		}
		decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
		if err != nil || len(decodedBytes) > 6*1024*1024 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  "{fileData:'',fileUploading:false}",
				IsSignal: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: "File too large. Please upload a file smaller than 6 MB.",
				IsError: true,
			}
			fileName = ""
			return
		}
		bytesBuffer := new(bytes.Buffer)
		components.FileAttachmentDisplay(fileName).Render(context.Background(), bytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "{fileUploading:false}",
			IsSignal: true,
		}
	}
}
func removeUploadedFileHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	bytesBuffer := new(bytes.Buffer)

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		components.FileAttachmentDisplay("").Render(context.Background(), bytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           bytesBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "{fileData:'',fileUploading:false}",
			IsSignal: true,
		}
	}
}

// ALGO
// Handle unauthorized user - user does not exist in table or session id coming from client is not valid
// Handle bad request - more than 1 file uploaded, invalid file(non pdf and non image), file size > 6 MB
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
		if err != nil || len(decodedBytes) > 6*1024*1024 {
			http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)

	if clientSignal.Prompt != "" {
		if clientSignal.SessionId == 0 {
			insertChatSessionChannel := make(chan int)
			defer close(insertChatSessionChannel)
			newSession := models.ChatSession{Title: clientSignal.Prompt}
			go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
			newSession.Id = <-insertChatSessionChannel
			clientSignal.SessionId = newSession.Id
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				if clientSignal.SessionId == 0 {
					userSession.(chan models.LongSSEData) <- models.LongSSEData{
						Content: `Failed to create new chat session. Please try again later.`,
						IsError: true,
					}
					return
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  `window.history.replaceState({},'','/` + strconv.Itoa(clientSignal.SessionId) + `')`,
					IsScript: true,
				}
				menuItemBuffer := new(bytes.Buffer)
				components.MenuItem(newSession, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
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

		if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
			if userMessageChat.Id == 0 {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: `Failed to save chat conversation. Please try again later.`,
					IsError: true,
				}
				return
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `document.getElementById('hint')?.remove();`,
				IsScript: true,
			}
			userMessageBuffer := new(bytes.Buffer)
			components.ChatMessage(userMessageChat, true).Render(context.Background(), userMessageBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           userMessageBuffer.String(),
				UseViewTransition: true,
				Mode:              datastar.WithModeAppend(),
				Selector:          "section",
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `{prompt:"",sessionId:` + strconv.Itoa(clientSignal.SessionId) + `,fileData:''}`,
				IsSignal: true,
			}
			fileAttachmentBuffer := new(bytes.Buffer)
			components.FileAttachmentDisplay("").Render(context.Background(), fileAttachmentBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           fileAttachmentBuffer.String(),
				UseViewTransition: true,
			}
		}
		markdownToHtmlChannel := make(chan string)
		go services.ConvertConversationMarkdownsToHtml([]models.ChatConversation{userMessageChat}, markdownToHtmlChannel)
		if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
			if userMessageChat.FileData != "" {
				chatMessageUserFileDataBuffer := new(bytes.Buffer)
				components.ChatMessageFileData(userMessageChat, true).Render(context.Background(), chatMessageUserFileDataBuffer)
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:           chatMessageUserFileDataBuffer.String(),
					UseViewTransition: true,
				}
			}

			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           <-markdownToHtmlChannel,
				UseViewTransition: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
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
	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		for _, id := range deletedIds {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
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

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           modelMessageChatBuffer.String(),
			UseViewTransition: true,
			Mode:              datastar.WithModeAppend(),
			Selector:          "section",
		}

		if modelMessageChat.Id == 0 {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  "Error storing chat conversation.",
				IsScript: true,
			}
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
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

	if len(openRouterRequest.Messages) == 2 {
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
	userSession, userSessionExists := uiSidMap.Load(userSessionKey)
	for msg := range openRouterChannel {
		if msg.DeltaContent == "Error" {
			fmt.Printf("Error in getting response from OpenRouter\n")
			// handle error
			continue
		}
		modelMessageChat.Content += msg.DeltaContent
		modelMessageChat.FileData += msg.DeltaImage
		modelMessageChat.ModelId = msg.ModelId
		if !userSessionExists {
			continue
		}
		// sse.PatchElementTempl(components.ChatMessage(modelMessageChat, true))
		markdownToHtmlChannel := make(chan string)
		go services.ConvertConversationMarkdownsToHtml([]models.ChatConversation{modelMessageChat}, markdownToHtmlChannel)
		if modelMessageChat.FileData != "" {
			modelMessageChat.FileName = "generated_image.png"
			modelMessageChatFileDataBuffer := new(bytes.Buffer)
			components.ChatMessageFileData(modelMessageChat, true).Render(context.Background(), modelMessageChatFileDataBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           modelMessageChatFileDataBuffer.String(),
				UseViewTransition: true,
			}
		} else {
			modelMessageChat.FileName = ""
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: <-markdownToHtmlChannel,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  "window.mermaid.run()",
			IsScript: true,
		}
	}
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsRemove: true,
			Selector: "#thinkingMesssage",
		}
		chatMessageModelIdBuffer := new(bytes.Buffer)
		components.ChatMessageModelIdDisplay(modelMessageChat).Render(context.Background(), chatMessageModelIdBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           chatMessageModelIdBuffer.String(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			IsSignal: true,
			Content:  `{showErrorMessage:false}`,
		}
	}

	if updateTitleCalled {
		if <-updateTitleChannel != 0 {
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				menuItemBuffer := new(bytes.Buffer)
				components.MenuItem(models.ChatSession{Id: clientSignal.SessionId, Title: titleToUpdate}, clientSignal.SearchMenu).Render(context.Background(), menuItemBuffer)
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
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
		if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
			if <-deleteModelConversationChannel != 0 {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					IsRemove: true,
					Selector: "#message-" + strconv.Itoa(modelMessageChat.Id),
				}
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
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
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists && rowsAffected == 0 {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: "Failed to update chat conversation. Please try again later.",
			IsError: true,
		}
	}
}
