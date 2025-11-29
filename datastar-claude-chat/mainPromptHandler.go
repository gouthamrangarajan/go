package main

import (
	"datastar-claude-chat/components"
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// ALGO
// all validation/error messages stops the flow except for claude message error
// Check for invalid/bad request : prompt empty , invalid session id, invalid image/pdf, invalid size & send bad request
// If session id is 0 in incoming request , insert new chat session
// generate fileData for image/pdf so that they can be stored in db , if not image/pdf , make this field empty
// Check if the session id is not part of user sessions, send unauthorized if so
// if unable to generate claude request , send internal server error (prompt + uploaded image/pdf data + allowWebsearch)
// append new chat session UI && window url replace using data star sse if insert new chat session was successful
// send error message using data star sse if insert new chat session has failed
// insert chat conversation from user after populating imgdata/pdfdata if needed and send error message via data star sse if failed
// send message template for user to append to UI via data star sse
// clear the prompt signal, file data signal, file data UI & scroll the user message into view using data star sse
// if the request is first for the session call session title update via channel
// call session allow web search flag update via channel
// insert chat conversation for assistant and send error message via data star sse if failed
// call claude api with channel to get streaming string output
// send message template for assistant to append to UI via data star sse
// range over channel , as long as its not closed , read the message string from channel
// consolidate the message , keep patching the assistant message UI with consolidate message for every loop iteration
// if there was at least one message with "Error" , dont consolidate this string and send error message via data star sse
// wait for session title update to be completed if called & if success send patchelement to ui via data star sse
// if only one response from claude api call and thats error then delete the message
// otherwise update the assistant message to db & wait
// wait for allow web search flag update to complete

func promptHandler(responseWriter http.ResponseWriter, request *http.Request) {

	requestBody, err := io.ReadAll(request.Body)

	var clientSignal models.ClientSignals
	err = json.Unmarshal(requestBody, &clientSignal)
	prompt := clientSignal.Prompt

	// fmt.Println(clientSignal)
	// http.Error(responseWriter, "Bad request", http.StatusBadRequest)
	// return

	fileData := ""
	fileDataForDb := ""
	fileMediaType := ""
	fileName := ""
	var imgMatches []string
	var pdfMatches []string
	if len(clientSignal.FileData) > 0 {
		fileDataForDb = "data:" + clientSignal.FileData[0].Mime + ";base64," + clientSignal.FileData[0].Contents
		fileData = clientSignal.FileData[0].Contents
		fileMediaType = clientSignal.FileData[0].Mime
		imgMatches = services.ImgRegex.FindStringSubmatch(fileDataForDb)
		pdfMatches = services.PdfRegex.FindStringSubmatch(fileDataForDb)
		fileName = clientSignal.FileData[0].Name
	}

	userId := request.Context().Value(services.UserIDKey).(string)
	sessionIdStr := request.URL.Query().Get("sessionId")
	sessionId, err := strconv.Atoi(sessionIdStr)
	var decodedBytes []byte
	var fileDataDecodeErr error
	if len(clientSignal.FileData) > 0 {
		decodedBytes, fileDataDecodeErr = base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
	}

	newSessionInserted := false
	if prompt == "" || err != nil ||
		fileDataDecodeErr != nil || len(decodedBytes) > 1024*1024 || len(clientSignal.FileData) > 1 ||
		(len(clientSignal.FileData) == 1 && len(imgMatches) != 4 && len(pdfMatches) != 2) {
		http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		return
	} else if sessionId == 0 {
		newSessionChannel := make(chan int)

		go services.InsertChatSession(userId, models.ChatSession{Title: prompt, AllowWebSearch: clientSignal.SearchWeb}, newSessionChannel)
		sessionId = <-newSessionChannel
		newSessionInserted = true
	}
	if len(imgMatches) != 4 &&
		len(pdfMatches) != 2 && fileData != "" {
		fileData = ""
		fileDataForDb = ""
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

	claudeRequest, errMsg := services.GenerateClaudeRequest(userId, models.PromptRequest{SessionId: sessionId, Prompt: prompt, FileMediaType: fileMediaType, FileData: fileData, SearchWeb: clientSignal.SearchWeb})

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
			sse.ExecuteScript("window.history.replaceState({},document.title,window.location.origin+'/'+" + strconv.Itoa(sessionId) + ");  ")
		} else {
			services.SendErrorMessageToUI(sse, "'Failed to save conversation. Please try again later.'")
			return
		}
	}

	userMessageInsertDbChannel := make(chan int)
	defer close(userMessageInsertDbChannel)
	userChatConversationToInsert := models.ChatConversation{SessionId: sessionId, Message: prompt,
		ImgData: "", PdfData: "", FileId: "", FileName: fileName, Sender: "user"}
	if fileDataForDb != "" {
		if len(imgMatches) == 4 {
			userChatConversationToInsert.ImgData = fileDataForDb
		} else if len(pdfMatches) == 2 {
			userChatConversationToInsert.PdfData = fileDataForDb
		}
	}
	go services.InsertChatConversation(userChatConversationToInsert, userMessageInsertDbChannel)
	userChatConversationToInsert.Id = <-userMessageInsertDbChannel

	if userChatConversationToInsert.Id == 0 {
		//error handling
		services.SendErrorMessageToUI(sse, "Failed to save conversation. Please try again later.")
		return
	}

	sse.PatchElementTempl(components.MessageForStreaming(userChatConversationToInsert), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	sse.PatchSignals([]byte("{prompt:''}"))
	if fileData != "" {
		sse.PatchSignals([]byte("{fileData:''}"))
		sse.PatchElementTempl(components.FileDataDisplay(models.FileDataDisplay{}, false), datastar.WithUseViewTransitions(true))
	}
	sse.ExecuteScript(`document.getElementById('messageContainer_`+strconv.Itoa(userChatConversationToInsert.Id)+`').scrollIntoView()`, datastar.WithExecuteScriptAutoRemove(true))

	isSessionTitleUpdate := false
	sessionTitleUpdateChannel := make(chan int)
	defer close(sessionTitleUpdateChannel)
	if len(claudeRequest.Messages) == 1 {
		go services.UpdateChatSessionTitle(userId, models.ChatSession{Id: sessionId, Title: prompt}, sessionTitleUpdateChannel)
		isSessionTitleUpdate = true
	}

	sessionSearchWebChannel := make(chan int)
	defer close(sessionSearchWebChannel)
	go services.UpdateChatSessionAllowWebSearch(userId, sessionId, clientSignal.SearchWeb, sessionSearchWebChannel)

	claudeResponseInsertDbChannel := make(chan int)
	defer close(claudeResponseInsertDbChannel)

	go services.InsertChatConversation(models.ChatConversation{SessionId: sessionId, Message: "", ImgData: "", FileId: "", FileName: "", Sender: "assistant"}, claudeResponseInsertDbChannel)
	claudeMessageId := <-claudeResponseInsertDbChannel
	if claudeMessageId == 0 {
		//error handling
		services.SendErrorMessageToUI(sse, "'Failed to save conversation. Please try again later.'")
		return
	}

	claudeResponseChannel := make(chan string)
	go services.CallClaudeAPI(claudeRequest, claudeResponseChannel)

	sse.PatchElementTempl(components.MessageForStreaming(models.ChatConversation{Id: claudeMessageId, Message: "", FileName: "", ImgData: "", Sender: "assistant"}), datastar.WithModeAppend(), datastar.WithSelectorID("messages"))
	mergedOutput := ""

	errored := false
	for response := range claudeResponseChannel {
		if response == "Error" {
			sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'An error occurred with the AI model. Please try again later.'}"))
			errored = true
		} else {
			mergedOutput += response
			sse.PatchElementTempl(components.MessageForStreaming(models.ChatConversation{Id: claudeMessageId, Message: mergedOutput, FileName: "", ImgData: "", Sender: "assistant"}))
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
	<-sessionSearchWebChannel
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
