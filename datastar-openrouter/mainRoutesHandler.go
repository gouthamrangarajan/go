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

	chatConversationChannel := make(chan []models.ChatConversation)
	defer close(chatConversationChannel)
	go services.GetChatConversations(userId, sessionId, chatConversationChannel)

	aiModelsChannel := make(chan []models.AIModel)
	defer close(aiModelsChannel)
	go services.GetAiModels(aiModelsChannel)

	chatConversations := <-chatConversationChannel
	aiModels := <-aiModelsChannel
	component := components.Main(chatConversations, sessions, aiModels, sessionId)
	templ.Handler(component).ServeHTTP(responseWriter, request)
}

func newChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := services.GetUserIdFromRequest(request)
	userExistsChannel := make(chan bool)
	defer close(userExistsChannel)
	go services.CheckUserExistsInTable(userId, userExistsChannel)
	if !<-userExistsChannel {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	}
	insertChatSessionChannel := make(chan int)
	defer close(insertChatSessionChannel)
	newSession := models.ChatSession{Title: "New Chat"}
	go services.InsertChatSession(userId, newSession, insertChatSessionChannel)
	newSession.Id = <-insertChatSessionChannel
	sse := datastar.NewSSE(responseWriter, request)
	if newSession.Id != 0 {
		sse.ExecuteScript(`window.location.href=window.location.origin+'/'+` + strconv.Itoa(newSession.Id))
	} else {
		services.SendErrorMessageToUI(sse, "Error creating new chat session. Please try again later.")
	}
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
			if clientSignal.SessionId == 0 {
				services.SendErrorMessageToUI(sse, "Error creating new chat session. Please try again later.")
				return
			}
			sse.ExecuteScript(`window.history.replaceState({},'','/`+strconv.Itoa(clientSignal.SessionId)+`')`, datastar.WithExecuteScriptAutoRemove(true))
			sse.PatchElementTempl(components.MenuItem(newSession), datastar.WithModeAppend(), datastar.WithSelector("#menu"))
		}

		insertUserConversationChannel := make(chan int)
		defer close(insertUserConversationChannel)
		userMessageChat := models.ChatConversation{Role: "user", Content: clientSignal.Prompt, SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(userMessageChat, insertUserConversationChannel)
		userMessageChat.Id = <-insertUserConversationChannel

		if userMessageChat.Id == 0 {
			services.SendErrorMessageToUI(sse, "Error storing chat conversation. Please try again later.")
			return
		}

		sse.ExecuteScript(`document.getElementById('hint')?.remove();`, datastar.WithExecuteScriptAutoRemove(true))
		sse.PatchElementTempl(components.ChatMessage(userMessageChat), datastar.WithModeAppend(), datastar.WithSelector("section"), datastar.WithUseViewTransitions(true))
		sse.PatchSignals([]byte(`{prompt:"",sessionId:` + strconv.Itoa(clientSignal.SessionId) + `}`))

		insertModelConversationChannel := make(chan int)
		defer close(insertModelConversationChannel)
		modelMessageChat := models.ChatConversation{Role: "assistant", Content: "", SessionId: clientSignal.SessionId}
		go services.InsertChatConversation(modelMessageChat, insertModelConversationChannel)
		modelMessageChat.Id = <-insertModelConversationChannel
		sse.PatchElementTempl(components.ChatMessage(modelMessageChat), datastar.WithModeAppend(), datastar.WithSelector("section"), datastar.WithUseViewTransitions(true))
		if modelMessageChat.Id == 0 {
			services.SendErrorMessageToUI(sse, "Error storing chat conversation")
		}

		sse.ExecuteScript(`document.querySelector("main").scrollTo(0, document.querySelector("main").scrollHeight);`, datastar.WithExecuteScriptAutoRemove(true))
		openRouterChannel := make(chan models.OpenRouterModelIdAndDeltaString)
		openRouterRequest, _ := services.GenerateOpenRouterRequest(userId, clientSignal)
		go services.CallOpenRouterWithStreaming(openRouterRequest, openRouterChannel)

		updateTitleChannel := make(chan int)
		defer close(updateTitleChannel)
		updateTitleCalled := false

		if len(openRouterRequest.Messages) == 1 {
			go services.UpdateChatSessionTitle(userId, models.ChatSession{Id: clientSignal.SessionId, Title: clientSignal.Prompt}, updateTitleChannel)
			updateTitleCalled = true
		}

		for msg := range openRouterChannel {
			if msg.DeltaContent == "Error" {
				fmt.Printf("Error in getting response from OpenRouter\n")
				// handle error
				services.SendErrorMessageToUI(sse, "Error in getting response from AI. Please try again later or try different model.")
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
		if updateTitleCalled {
			<-updateTitleChannel
			select {
			case <-request.Context().Done():
				break
			default:
				sse.PatchElementTempl(components.MenuItem(models.ChatSession{Id: clientSignal.SessionId, Title: clientSignal.Prompt}))
			}
		}
		if modelMessageChat.Content != "" && modelMessageChat.Id != 0 {
			updateModelConversationChannel := make(chan int)
			defer close(updateModelConversationChannel)
			go services.UpateMessageChatConversation(models.UpdateChatConversation{
				Id:      modelMessageChat.Id,
				Content: modelMessageChat.Content,
				ModelId: modelMessageChat.ModelId,
			}, updateModelConversationChannel)
			rowsAffected := <-updateModelConversationChannel
			if rowsAffected == 0 {
				services.SendErrorMessageToUI(sse, "Error updating chat conversation")
			}
		}
	}
}
