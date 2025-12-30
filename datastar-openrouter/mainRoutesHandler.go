package main

import (
	"datastar-openrouter/components"
	"datastar-openrouter/models"
	"datastar-openrouter/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionId := 0
	sessionIdStr := chi.URLParam(request, "sessionId")
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
	// sessionId := 0
	// if len(sessions) == 0 {
	// 	insertChatSessionChannel := make(chan int)
	// 	defer close(insertChatSessionChannel)
	// 	go services.InsertChatSession(userId, models.ChatSession{Title: "New Chat"}, insertChatSessionChannel)
	// 	sessionId = <-insertChatSessionChannel
	// } else {
	// 	sessionId = sessions[0].Id
	// }
	chatConversationChannel := make(chan []models.ChatConversation)
	defer close(chatConversationChannel)
	go services.GetChatConversations(userId, sessionId, chatConversationChannel)
	chatConversations := <-chatConversationChannel
	component := components.Main(chatConversations, sessionId)
	templ.Handler(component).ServeHTTP(responseWriter, request)
}

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := services.GetUserIdFromRequest(request)
	requestBody, _ := io.ReadAll(request.Body)
	var clientSignal models.ClientSignals
	json.Unmarshal(requestBody, &clientSignal)
	clientSignal.Prompt = strings.TrimSpace(clientSignal.Prompt)

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

	if clientSignal.Prompt != "" {
		sse := datastar.NewSSE(responseWriter, request)
		if clientSignal.SessionId == 0 {
			insertChatSessionChannel := make(chan int)
			defer close(insertChatSessionChannel)
			newSession := models.ChatSession{Title: clientSignal.Prompt}
			go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
			newSession.Id = <-insertChatSessionChannel
			clientSignal.SessionId = newSession.Id
			sse.ExecuteScript(`window.history.replaceState({},'','/`+strconv.Itoa(clientSignal.SessionId)+`')`, datastar.WithExecuteScriptAutoRemove(true))
		}

		insertUserConversationChannel := make(chan int)
		defer close(insertUserConversationChannel)
		userMessageChat := models.ChatConversation{Role: "user", Content: clientSignal.Prompt, SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(userMessageChat, insertUserConversationChannel)
		userMessageChat.Id = <-insertUserConversationChannel

		sse.ExecuteScript(`document.getElementById('hint')?.remove();`, datastar.WithExecuteScriptAutoRemove(true))
		sse.PatchElementTempl(components.ChatMessage(userMessageChat), datastar.WithModeAppend(), datastar.WithSelector("main"), datastar.WithUseViewTransitions(true))
		sse.PatchSignals([]byte(`{prompt:"",sessionId:` + strconv.Itoa(clientSignal.SessionId) + `}`))

		insertModelConversationChannel := make(chan int)
		defer close(insertModelConversationChannel)
		modelMessageChat := models.ChatConversation{Role: "assistant", Content: "", SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(modelMessageChat, insertModelConversationChannel)
		modelMessageChat.Id = <-insertModelConversationChannel
		sse.PatchElementTempl(components.ChatMessage(modelMessageChat), datastar.WithModeAppend(), datastar.WithSelector("main"), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript(`document.querySelector("main").scrollTo(0, document.querySelector("main").scrollHeight);`, datastar.WithExecuteScriptAutoRemove(true))
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
