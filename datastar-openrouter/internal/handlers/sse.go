package handlers

import (
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components"
	"datastar-openrouter/services"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

func LongSSEHandler(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(services.UserIDKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	// fmt.Printf("uisid from client %v\n", clientSignal.UiSid)

	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)

	userSessionChannel := make(chan models.LongSSEData, 16)
	uiSidMap.Store(userSessionKey, userSessionChannel)
	defer uiSidMap.CompareAndDelete(userSessionKey, userSessionChannel)

	if clientSignal.SessionId != 0 {
		go sendConversationsMarkdown(clientSignal, userId)
	}

	responseWriter.Header().Set("Content-Type", "text/event-stream")
	responseWriter.Header().Set("Cache-Control", "no-cache")
	responseWriter.Header().Set("Connection", "keep-alive")
	// This header tells Nginx/Railway Proxy not to buffer the stream
	responseWriter.Header().Set("X-Accel-Buffering", "no")

	sse := datastar.NewSSE(responseWriter, request)

	liveIndicatorTicker := time.NewTicker(5 * time.Second)
	defer liveIndicatorTicker.Stop()

	sse.PatchSignals([]byte(`{showErrorMessage:false}`))
	for {
		select {
		case <-request.Context().Done():
			return
		case data := <-userSessionChannel:
			if channelInMap, ok := uiSidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			switch {
			case data.SendHeartBeat:
				sse.Send(datastar.EventType("heartbeat"), []string{fmt.Sprintf(": heartbeat %d\n\n", time.Now().Unix())})
			case data.IsError:
				services.SendErrorMessageToUI(sse, data.Content)
			case data.IsScript:
				sse.ExecuteScript(data.Content, datastar.WithExecuteScriptAutoRemove(true))
			case data.IsSignal:
				sse.PatchSignals([]byte(data.Content))
			case data.IsRemove:
				sse.RemoveElement(data.Selector, datastar.WithUseViewTransitions(data.UseViewTransition))
			default:
				if strings.TrimSpace(data.Selector) == "" {
					sse.PatchElements(data.Content, datastar.WithUseViewTransitions(data.UseViewTransition))
				} else if data.Mode != nil {
					sse.PatchElements(data.Content, datastar.WithSelector(data.Selector), data.Mode, datastar.WithUseViewTransitions(data.UseViewTransition))
				} else {
					sse.PatchElements(data.Content, datastar.WithSelector(data.Selector), datastar.WithUseViewTransitions(data.UseViewTransition))
				}
			}

		case <-liveIndicatorTicker.C:
			if channelInMap, ok := uiSidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			sse.PatchElementTempl(components.LiveIndicator(), datastar.WithUseViewTransitions(false))
		}
	}
}

func sendConversationsMarkdown(clientSignal models.ClientSignals, userId string) {
	userSessionKey := services.GenerateUserSessionKey(userId, clientSignal.UiSid)
	conversationsChannel := make(chan []models.ChatConversation)

	go services.GetChatConversationsWithoutFileData(userId, clientSignal.SessionId, conversationsChannel)
	conversations := <-conversationsChannel
	defer close(conversationsChannel)

	if len(conversations) != 0 {
		markdownToHtmlChannel := make(chan models.ChatConversationMarkdownToHtml)
		go services.ConvertConversationMarkdownsToHtml(conversations, markdownToHtmlChannel)

		for element := range markdownToHtmlChannel {
			if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content: element.Html,
				}
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:  `window.mermaid.run()`,
					IsScript: true,
				}
			}
		}
	}
	if userSession, userSessionExists := uiSidMap.Load(userSessionKey); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  `{pageLoading:false}`,
			IsSignal: true,
		}
	}
}
