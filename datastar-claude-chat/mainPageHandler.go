package main

import (
	"datastar-claude-chat/components"
	"datastar-claude-chat/components/shared"
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func mainPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sessionId := 0
	allowWebSearch := false
	sessionIdStr := chi.URLParam(request, "sessionId")
	sessionId, err := strconv.Atoi(sessionIdStr)
	if err != nil {
		sessionId = 0
	}
	if sessionId == 0 {
		components.Main(0, allowWebSearch, []models.ChatConversation{}).Render(request.Context(), responseWriter)
		return
	}
	userId := request.Context().Value(services.UserIDKey).(string)

	sessionsChannel := make(chan []models.ChatSession)
	defer close(sessionsChannel)
	go services.GetChatSessions(userId, sessionsChannel)
	sessions := <-sessionsChannel
	sessionFound := false

	for _, session := range sessions {
		if session.Id == sessionId {
			allowWebSearch = session.AllowWebSearch
			sessionFound = true
			break
		}
	}
	if !sessionFound {
		http.Error(responseWriter, "UnAuthorized", http.StatusUnauthorized)
		return
	}
	conversationChannel := make(chan []models.ChatConversation)
	defer close(conversationChannel)
	go services.GetChatConversations(userId, sessionId, conversationChannel)
	conversations := <-conversationChannel
	components.Main(sessionId, allowWebSearch, conversations).Render(request.Context(), responseWriter)
}

func menuDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	sessions := services.GetChatSessionsViaChannel(userId)
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.ChatSessionMenuItems(sessions), datastar.WithModeInner(), datastar.WithSelectorID("menuContainer"), datastar.WithUseViewTransitions(true))
}

func newChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	newSessionId := services.InsertChatSessionViaChannel(userId, models.ChatSession{Title: "New Chat", AllowWebSearch: false})
	sse := datastar.NewSSE(responseWriter, request)
	if newSessionId == 0 {
		sse.PatchSignals([]byte("{showErrorMessage:true,errorMessage:'Failed to create new conversation. Please try again later.'}"))
		time.Sleep(3000 * time.Millisecond)
		sse.PatchSignals([]byte("{showErrorMessage:false}"))
		return
	}
	sse.PatchSignals([]byte("{showMenu:false}"))
	time.Sleep(200 * time.Millisecond)
	sse.ExecuteScript("window.location.href=window.location.origin+'/'+" + strconv.Itoa(newSessionId))
}

func fileuploadHandler(responseWriter http.ResponseWriter, request *http.Request) {
	requestBody, _ := io.ReadAll(request.Body)
	var clientSignal models.ClientSignals

	_ = json.Unmarshal(requestBody, &clientSignal)

	fileDataForUI := ""
	fileName := ""
	if len(clientSignal.FileData) > 0 {
		fileDataForUI = "data:" + clientSignal.FileData[0].Mime + ";base64," + clientSignal.FileData[0].Contents
		fileName = clientSignal.FileData[0].Name
	}

	sse := datastar.NewSSE(responseWriter, request)
	imgMatches := services.ImgRegex.FindStringSubmatch(fileDataForUI)
	pdfMatches := services.PdfRegex.FindStringSubmatch(fileDataForUI)

	if len(clientSignal.FileData) == 0 {
		fileName = ""
		fileDataForUI = ""
	} else if len(clientSignal.FileData) > 1 {
		sse.PatchSignals([]byte("{fileData:''}"))
		services.SendErrorMessageToUI(sse, "Please select only one file at a time.")
		fileName = ""
		fileDataForUI = ""
	} else if (clientSignal.FileData[0].Mime == "application/pdf" && len(pdfMatches) != 2) ||
		(clientSignal.FileData[0].Mime != "application/pdf" && len(imgMatches) != 4) {
		sse.PatchSignals([]byte("{fileData:''}"))
		services.SendErrorMessageToUI(sse, "Invalid file type. Please upload an file with type (JPG, PNG, WEBP, GIF, PDF)")
		fileName = ""
		fileDataForUI = ""
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.FileData[0].Contents)
	if err != nil || len(decodedBytes) > 1024*1024 {
		sse.PatchSignals([]byte("{fileData:''}"))
		services.SendErrorMessageToUI(sse, "File size exceeds the limit of 1 MB")
		fileName = ""
		fileDataForUI = ""
	}
	if fileDataForUI != "" {
		sse.PatchElementTempl(components.FileDataDisplay(models.FileDataDisplay{FileData: fileDataForUI, FileName: fileName}, len(imgMatches) == 4), datastar.WithUseViewTransitions(true))
	}
}

func deleteChatHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	requestBody, _ := io.ReadAll(request.Body)
	var clientSignal models.ClientSignals

	_ = json.Unmarshal(requestBody, &clientSignal)
	channel := make(chan int)
	defer close(channel)
	go services.DeleteChatSession(userId, clientSignal.SessionIdToDelete, channel)
	rowsAffected := <-channel
	sse := datastar.NewSSE(responseWriter, request)
	if rowsAffected != 1 {
		sse.PatchElementTempl(shared.DeleteErrorMessage("Failed to delete the chat session. Please try again later."))
		return
	}
	sse.RemoveElement("#menu_"+strconv.Itoa(clientSignal.SessionIdToDelete), datastar.WithUseViewTransitions(true))
	sse.PatchSignals([]byte("{showDeleteModal:false}"))
	if clientSignal.SessionId == clientSignal.SessionIdToDelete {
		sse.PatchSignals([]byte("{sessionId:0}"))
		sse.PatchElementTempl(components.MessagesSection([]models.ChatConversation{}))
		sse.ExecuteScript("window.history.replaceState({},document.title,window.location.origin);")
	}
}
