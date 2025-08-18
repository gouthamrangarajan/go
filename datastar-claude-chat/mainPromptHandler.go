package main

import (
	"datastar-claude-chat/components"
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {
	prompt := request.FormValue("prompt")
	if prompt == "" {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	}
	userId := request.Context().Value(services.UserIDKey).(string)
	if userId == "" {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	}
	sessionIdStr := request.URL.Query().Get("sessionId")
	sessionId, err := strconv.Atoi(sessionIdStr)
	newSessionInserted := false
	if err != nil || sessionIdStr == "" {
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	} else if sessionId == 0 {
		newSessionChannel := make(chan int)
		go services.InsertChatSession(userId, prompt, newSessionChannel)
		sessionId = <-newSessionChannel
		newSessionInserted = true
	}

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel

	sessionFound := false
	for _, session := range sessions {
		if session.Id == sessionId {
			sessionFound = true
			break
		}
	}
	if !sessionFound {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}

	claudeRequest, errMsg := services.GenerateClaudeRequest(userId, sessionId, prompt)

	if errMsg != "" {
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(responseWriter, request)
	if newSessionInserted {
		chatSession := models.ChatSession{
			Id:    sessionId,
			Title: prompt,
		}
		sse.PatchSignals([]byte("{sessionId:" + strconv.Itoa(sessionId) + "}"))
		sse.PatchElementTempl(components.ChatSessionMenuItems(append([]models.ChatSession{}, chatSession)), datastar.WithModeAppend(), datastar.WithSelectorID("menuContainer"), datastar.WithUseViewTransitions(true))
	}

	userMessageInsertDbChannel := make(chan int)
	defer close(userMessageInsertDbChannel)
	go services.InsertChatConversation(sessionId, prompt, "", "user", userMessageInsertDbChannel)
	userMessageId := <-userMessageInsertDbChannel

	if userMessageId == 0 {
		//error handling
		sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'Failed to save conversation. Please try again later.'}"))
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
		return
	}
	sse.PatchElementTempl(components.MessageForStreaming(userMessageId, prompt, "user"), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	sse.PatchSignals([]byte("{prompt:''}"))
	sse.ExecuteScript(`document.getElementById('messageContainer_`+strconv.Itoa(userMessageId)+`').scrollIntoView()`, datastar.WithExecuteScriptAutoRemove(true))

	claudeResponseInsertDbChannel := make(chan int)
	defer close(claudeResponseInsertDbChannel)
	go services.InsertChatConversation(sessionId, "", "", "assistant", claudeResponseInsertDbChannel)
	claudeMessageId := <-claudeResponseInsertDbChannel
	if claudeMessageId == 0 {
		//error handling
		sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'Failed to save conversation. Please try again later.'}"))
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
		return
	}

	claudeResponseChannel := make(chan string)
	go services.CallClaudeAPIStreamingWithRequest(claudeRequest, claudeResponseChannel)

	sse.PatchElementTempl(components.MessageForStreaming(claudeMessageId, "", "assistant"), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	mergedOutput := ""

	errored := false
	for response := range claudeResponseChannel {
		if response == "Error" {
			sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'An error occurred with the AI model. Please try again later.'}"))
			errored = true
		}
		mergedOutput += response
		sse.PatchElementTempl(components.MessageForStreaming(claudeMessageId, mergedOutput, "assistant"))
	}
	if errored {
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
	}
	updateMessageChannel := make(chan int)
	defer close(updateMessageChannel)
	go services.UpateMessageChatConversation(claudeMessageId, mergedOutput, updateMessageChannel)
	<-updateMessageChannel
}

func getMockAPIResponse(channel chan string) {
	time.Sleep(2000 * time.Millisecond)
	defer close(channel)
	sample := "Sure! Here is a simple Go program that prints 'Hello, World!':\n\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```"
	for _, word := range strings.Split(sample, " ") {
		time.Sleep(25 * time.Millisecond)
		channel <- word + " "
	}
}
