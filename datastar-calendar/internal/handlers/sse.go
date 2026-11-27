package handlers

import (
	"datastar-calendar/internal/models"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type SSEHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewSSEHandler(dbService *db.DBService, uidSidMap *sync.Map) *SSEHandler {
	return &SSEHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (s *SSEHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	var signals models.ClientSignals
	datastar.ReadSignals(request, &signals)
	sessionKey := services.GenerateUserSessionKey(token, signals.UiSid)

	userSessionChannel := make(chan models.LongSSEData, 16)
	s.uiSidMap.Store(sessionKey, userSessionChannel)
	defer s.uiSidMap.CompareAndDelete(sessionKey, userSessionChannel)

	responseWriter.Header().Set("Content-Type", "text/event-stream")
	responseWriter.Header().Set("Cache-Control", "no-cache")
	responseWriter.Header().Set("Connection", "keep-alive")
	// This header tells Nginx/Railway Proxy not to buffer the stream
	responseWriter.Header().Set("X-Accel-Buffering", "no")

	sse := datastar.NewSSE(responseWriter, request)

	heartBeatTicker := time.NewTicker(5 * time.Second)
	defer heartBeatTicker.Stop()

	sse.PatchSignals([]byte(`{showErrorMessage:false}`))
	for {
		select {
		case <-request.Context().Done():
			return
		case channelData := <-userSessionChannel:
			if channelInMap, ok := s.uiSidMap.Load(sessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			switch {
			case channelData.IsRemove:
				sse.RemoveElement(channelData.Selector, datastar.WithUseViewTransitions(channelData.UseViewTransition))
			case channelData.IsScript:
				sse.ExecuteScript(channelData.Content, datastar.WithExecuteScriptAutoRemove(true))
			case channelData.IsSignals:
				sse.PatchSignals([]byte(channelData.Content))
			default:
				if strings.TrimSpace(channelData.Selector) == "" {
					sse.PatchElements(channelData.Content, datastar.WithUseViewTransitions(channelData.UseViewTransition))
				} else if channelData.Mode != nil {
					sse.PatchElements(channelData.Content, datastar.WithSelector(channelData.Selector), channelData.Mode, datastar.WithUseViewTransitions(channelData.UseViewTransition))
				} else {
					sse.PatchElements(channelData.Content, datastar.WithSelector(channelData.Selector), datastar.WithUseViewTransitions(channelData.UseViewTransition))
				}
			}
		case <-heartBeatTicker.C:
			if channelInMap, ok := s.uiSidMap.Load(sessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			sse.Send(datastar.EventType("heartbeat"), []string{fmt.Sprintf(": heartbeat %d\n\n", time.Now().Unix())})
		}
	}
}
