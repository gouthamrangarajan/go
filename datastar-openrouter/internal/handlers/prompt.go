package handlers

import (
	"bytes"
	"context"
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components"
	"datastar-openrouter/services"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
)

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
func PromptHandler(responseWriter http.ResponseWriter, request *http.Request) {
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
		markdownToHtmlChannel := make(chan models.ChatConversationMarkdownToHtml)
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
			element := <-markdownToHtmlChannel
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           element.Html,
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
func RetryHandler(responseWriter http.ResponseWriter, request *http.Request) {
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
		markdownToHtmlChannel := make(chan models.ChatConversationMarkdownToHtml)
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
		element := <-markdownToHtmlChannel
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: element.Html,
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
