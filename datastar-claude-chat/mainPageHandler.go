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
	newSessionId := services.InsertChatSessionViaChannel(userId, "New Chat", false)
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

	imgData := ""
	imgName := ""
	if len(clientSignal.ImgData) > 0 && len(clientSignal.ImgMimes) > 0 {
		imgData = "data:" + clientSignal.ImgMimes[0] + ";base64," + clientSignal.ImgData[0]
	}
	if len(clientSignal.ImgNames) > 0 {
		imgName = clientSignal.ImgNames[0]
	}
	sse := datastar.NewSSE(responseWriter, request)
	matches := services.ImgRegex.FindStringSubmatch(imgData)
	if len(clientSignal.ImgData) > 1 {
		sse.PatchSignals([]byte("{imgData:''}"))
		services.SendErrorMessageToUI(sse, "Please select only one image at a time.")
		imgName = ""
		imgData = ""
	}
	if len(matches) != 4 {
		sse.PatchSignals([]byte("{imgData:''}"))
		services.SendErrorMessageToUI(sse, "Invalid file type. Please upload an image with type (JPG, PNG, WEBP)")
		imgName = ""
		imgData = ""
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(clientSignal.ImgData[0])
	if err != nil || len(decodedBytes) > 1024*1024 {
		sse.PatchSignals([]byte("{imgData:''}"))
		services.SendErrorMessageToUI(sse, "'Image size exceeds the limit of 1 MB'")
		imgName = ""
		imgData = ""
	}
	sse.PatchElementTempl(components.FileDataDisplay(imgData, imgName), datastar.WithUseViewTransitions(true))
}
