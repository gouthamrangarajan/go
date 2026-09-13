package handlers

import (
	"datastar-openrouter/internal/models"
	"datastar-openrouter/internal/views/components"
	"datastar-openrouter/services"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"sync"

	"github.com/starfederation/datastar-go/datastar"
)

type SSEHandler struct {
	uisidMap      *sync.Map
	userIdKey     string
	helperService *services.HelperService
}

func NewSSEHandler(uisidMap *sync.Map, helperService *services.HelperService) *SSEHandler {
	return &SSEHandler{
		uisidMap:      uisidMap,
		userIdKey:     os.Getenv("USER_ID_KEY"),
		helperService: helperService,
	}
}
func (s *SSEHandler) HandleLongSSE(responseWriter http.ResponseWriter, request *http.Request) {
	userId := request.Context().Value(s.userIdKey).(string)
	var clientSignal models.ClientSignals
	datastar.ReadSignals(request, &clientSignal)
	// fmt.Printf("uisid from client %v\n", clientSignal.UiSid)

	userSessionKey := s.helperService.GenerateUserSessionKey(userId, clientSignal.UiSid)

	userSessionChannel := make(chan models.LongSSEData, 16)
	s.uisidMap.Store(userSessionKey, userSessionChannel)
	defer s.uisidMap.CompareAndDelete(userSessionKey, userSessionChannel)

	if clientSignal.SessionId != 0 {
		go s.helperService.ConvertConversationMarkdownToHtmlAndSendToUserSessionChannel(s.uisidMap, clientSignal, userId)
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
			if channelInMap, ok := s.uisidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			switch {
			case data.SendHeartBeat:
				sse.Send(datastar.EventType("heartbeat"), []string{fmt.Sprintf(": heartbeat %d\n\n", time.Now().Unix())})
			case data.IsError:
				s.helperService.SendErrorMessageToUI(sse, data.Content)
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
			if channelInMap, ok := s.uisidMap.Load(userSessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			sse.PatchElementTempl(components.LiveIndicator(), datastar.WithUseViewTransitions(false))
		}
	}
}
