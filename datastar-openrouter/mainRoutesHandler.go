package main

import (
	"datastar-openrouter/components"
	"datastar-openrouter/models"
	"datastar-openrouter/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	sessionId := 0
	if len(sessions) == 0 {
		insertChatSessionChannel := make(chan int)
		defer close(insertChatSessionChannel)
		go services.InsertChatSession(userId, models.ChatSession{Title: "New Chat"}, insertChatSessionChannel)
		sessionId = <-insertChatSessionChannel
	} else {
		sessionId = sessions[0].Id
	}
	chatConversationChannel := make(chan []models.ChatConversation)
	defer close(chatConversationChannel)
	go services.GetChatConversations(userId, sessionId, chatConversationChannel)
	chatConversations := <-chatConversationChannel
	component := components.Main(chatConversations)
	templ.Handler(component).ServeHTTP(responseWriter, request)
}

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {
	requestBody, _ := io.ReadAll(request.Body)
	var clientSignal models.ClientSignals
	json.Unmarshal(requestBody, &clientSignal)
	userId := services.GetUserIdFromRequest(request)
	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	if clientSignal.Prompt != "" && len(sessions) > 0 {
		clientSignal.SessionId = sessions[0].Id
		insertUserConversationChannel := make(chan int)
		defer close(insertUserConversationChannel)
		userMessageChat := models.ChatConversation{Role: "user", Content: clientSignal.Prompt, SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(userMessageChat, insertUserConversationChannel)
		userMessageChat.Id = <-insertUserConversationChannel

		sse := datastar.NewSSE(responseWriter, request)
		sse.ExecuteScript(`document.getElementById('hint')?.remove();`)
		sse.PatchElementTempl(components.ChatMessage(userMessageChat), datastar.WithModeAppend(), datastar.WithSelector("main"), datastar.WithUseViewTransitions(true))
		sse.PatchSignals([]byte(`{prompt:""}`))

		insertModelConversationChannel := make(chan int)
		defer close(insertModelConversationChannel)
		modelMessageChat := models.ChatConversation{Role: "assistant", Content: "", SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(modelMessageChat, insertModelConversationChannel)
		modelMessageChat.Id = <-insertModelConversationChannel
		sse.PatchElementTempl(components.ChatMessage(modelMessageChat), datastar.WithModeAppend(), datastar.WithSelector("main"), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript(`document.querySelector("main").scrollTo(0, document.querySelector("main").scrollHeight);`)
		channel := make(chan models.OpenRouterModelIdAndDeltaString)
		openRouterRequest, _ := services.GenerateOpenRouterRequest(userId, clientSignal)
		go services.CallOpenRouterWithStreaming(openRouterRequest, channel)

		for msg := range channel {
			if msg.DeltaContent == "Error" {
				fmt.Printf("Error in getting response from OpenRouter\n")
				// handle error
				continue
			}
			modelMessageChat.Content += msg.DeltaContent
			modelMessageChat.ModelId = msg.ModelId
			select {
			case <-request.Context().Done():
				continue
			default:
				sse.PatchElementTempl(components.ChatMessage(modelMessageChat))
			}
		}
		updateModelConversationChannel := make(chan int)
		defer close(updateModelConversationChannel)
		go services.UpateMessageChatConversation(models.UpdateChatConversation{
			Id:      modelMessageChat.Id,
			Content: modelMessageChat.Content,
			ModelId: modelMessageChat.ModelId,
		}, updateModelConversationChannel)
		<-updateModelConversationChannel
	}
}
