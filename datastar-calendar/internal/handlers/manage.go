package handlers

import (
	"bytes"
	"context"
	"datastar-calendar/internal/models"
	"datastar-calendar/internal/views/components"
	"datastar-calendar/internal/views/pages"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"net/http"
	"strings"
	"sync"

	"github.com/starfederation/datastar-go/datastar"
)

type ManageEventsHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewManageEventsHandler(dbService *db.DBService, uidSidMap *sync.Map) *ManageEventsHandler {
	return &ManageEventsHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (m *ManageEventsHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	channel := make(chan []models.EventData)
	go m.dbService.GetAllNotOnlyOnceFrequencyData(token, channel)
	eventsData := <-channel
	pageData := services.ConstructManageEventsPageData(eventsData, models.ALL_STATUS)
	pages.ManageEvents(pageData).Render(request.Context(), responseWriter)
}

func (m *ManageEventsHandler) Filter(responseWriter http.ResponseWriter, request *http.Request) {
	if strings.ToUpper(request.Method) != "POST" {
		responseWriter.WriteHeader(405)
		return
	}

	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)

	channel := make(chan []models.EventData)
	go m.dbService.GetAllNotOnlyOnceFrequencyData(token, channel)
	eventsData := <-channel
	pageData := services.ConstructManageEventsPageData(eventsData, clientSignals.ManagePageFilter)

	if userSession, userSessionExists := m.uiSidMap.Load(key); userSessionExists {
		tableBytesBuffer := new(bytes.Buffer)
		components.ManageEventsTable(pageData.EventsData).Render(context.Background(), tableBytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: tableBytesBuffer.String(),
			Mode:    datastar.WithModeOuter(),
			// UseViewTransition: true,
		}
	}

}
