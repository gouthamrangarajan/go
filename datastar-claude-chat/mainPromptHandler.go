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

// ALGO
// all validation/error messages stops the flow except for claude message error
// Check for invalid/bad request : prompt empty , invalid session id, & send bad request
// If session id is 0 in incoming request , insert new chat session
// Check if the session id is not part of user sessions, send unauthorized if so
// if unable to generate claude request , send internal server error
// append new chat session UI using data star sse if insert new chat session was sucessful
// send error message using data star sse if insert new chat session has failed
// insert chat conversation from user and send error message via data star sse if failed
// send message template for user to append to UI via data star sse
// clear the prompt signal & scroll the user message into view using data star sse
// if the request is first for the session call session title update via channel
// insert chat conversation for assistant and send error message via data star sse if failed
// call claude api with channel to get streaming string output
// send message template for assistant to append to UI via data star sse
// range over channel , as long as its not closed , read the message string from channel
// consolidate the message , keep patching the assistant message UI with consolidate message for every loop iteration
// if there was at least one message with "Error" , dont consolidate this string and send error message via data star sse
// wait for session title update to be completed if called & if success send patchelement to ui via data star sse
// if only one response from claude api call and thats error then delete the message
// otherwise update the assistant message to db & wait

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {
	prompt := request.FormValue("prompt")
	userId := request.Context().Value(services.UserIDKey).(string)
	sessionIdStr := request.URL.Query().Get("sessionId")
	sessionId, err := strconv.Atoi(sessionIdStr)
	newSessionInserted := false

	if prompt == "" || err != nil {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
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
		if sessionId != 0 {
			chatSession := models.ChatSession{
				Id:    sessionId,
				Title: prompt,
			}
			sse.PatchSignals([]byte("{sessionId:" + strconv.Itoa(sessionId) + "}"))
			sse.PatchElementTempl(components.ChatSessionMenuItems(append([]models.ChatSession{}, chatSession)), datastar.WithModeAppend(), datastar.WithSelectorID("menuContainer"))
		} else {
			sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'Failed to save conversation. Please try again later.'}"))
			time.Sleep(3000 * time.Millisecond)
			sse.PatchSignals([]byte("{showErrorMessage:false}"))
			return
		}
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

	isSessionTitleUpdate := false
	sessionTitleUpdateChannel := make(chan int)
	defer close(sessionTitleUpdateChannel)
	if len(claudeRequest.Messages) == 1 {
		go services.UpdateChatSessionTitle(userId, sessionId, prompt, sessionTitleUpdateChannel)
		isSessionTitleUpdate = true
	}

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
	go services.CallClaudeAPI(claudeRequest, claudeResponseChannel)

	sse.PatchElementTempl(components.MessageForStreaming(claudeMessageId, "", "assistant"), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	mergedOutput := ""

	errored := false
	for response := range claudeResponseChannel {
		if response == "Error" {
			sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'An error occurred with the AI model. Please try again later.'}"))
			errored = true
		} else {
			mergedOutput += response
			sse.PatchElementTempl(components.MessageForStreaming(claudeMessageId, mergedOutput, "assistant"))
		}
	}
	if errored {
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
	}
	if isSessionTitleUpdate && <-sessionTitleUpdateChannel > 0 {
		chatSession := models.ChatSession{Id: sessionId, Title: prompt}
		sse.PatchElementTempl(components.ChatSessionMenuItems(append([]models.ChatSession{}, chatSession)))
	}
	updateOrDeleteMessageChannel := make(chan int)
	defer close(updateOrDeleteMessageChannel)
	if !errored || mergedOutput != "Error" {
		go services.UpateMessageChatConversation(claudeMessageId, mergedOutput, updateOrDeleteMessageChannel)
	} else {
		go services.DeleteClaudeMessageChatConversation(claudeMessageId, updateOrDeleteMessageChannel)
	}
	<-updateOrDeleteMessageChannel

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
