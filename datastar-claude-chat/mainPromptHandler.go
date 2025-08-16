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

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	sessionId := 0
	if len(sessions) == 0 {
		http.Error(responseWriter, "Unauthorized", http.StatusUnauthorized)
		return
	} else {
		sessionId = sessions[0].Id
	}

	claudeRequest, errMsg := services.GenerateClaudeRequest(userId, sessionId, prompt)

	if errMsg != "" {
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	claudeResponseChannel := make(chan string)
	go services.CallClaudeAPIStreamingWithRequest(claudeRequest, claudeResponseChannel)

	sse := datastar.NewSSE(responseWriter, request)

	userMessageInsertDbChannel := make(chan int)
	defer close(userMessageInsertDbChannel)
	go services.InsertChatConversation(sessionId, prompt, "", "user", userMessageInsertDbChannel)
	userMessageId := <-userMessageInsertDbChannel
	if userMessageId == 0 {
		//error handling
	}
	sse.PatchElementTempl(components.Message(userMessageId, prompt, "user"), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	sse.PatchSignals([]byte("{prompt:''}"))
	sse.ExecuteScript(`document.getElementById('messageContainer_`+strconv.Itoa(userMessageId)+`').scrollIntoView()`, datastar.WithExecuteScriptAutoRemove(true))

	claudeResponseInsertDbChannel := make(chan int)
	defer close(claudeResponseInsertDbChannel)
	go services.InsertChatConversation(sessionId, "", "", "assistant", claudeResponseInsertDbChannel)
	claudeMessageId := <-claudeResponseInsertDbChannel
	if claudeMessageId == 0 {
		//error handling
	}

	sse.PatchElementTempl(components.Message(claudeMessageId, "", "assistant"), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	mergedOutput := ""
	for response := range claudeResponseChannel {
		mergedOutput += response
		sse.PatchElementTempl(components.Message(claudeMessageId, mergedOutput, "assistant"))
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
