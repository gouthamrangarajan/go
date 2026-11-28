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
	pageData, totalRecords := services.ConstructManageEventsPageData(eventsData, models.ALL_STATUS, 1, "")
	pages.ManageEvents(pageData, totalRecords).Render(request.Context(), responseWriter)
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
	pageData, totalRecords := services.ConstructManageEventsPageData(eventsData,
		clientSignals.ManagePageFilter, clientSignals.ManagePageNumber,
		clientSignals.ManagePageSearchTxt)

	if userSession, userSessionExists := m.uiSidMap.Load(key); userSessionExists {
		tableBytesBuffer := new(bytes.Buffer)
		components.ManageEventsTable(pageData.EventsData, totalRecords).Render(context.Background(), tableBytesBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           tableBytesBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
		filterAllEventButtonBuffer := new(bytes.Buffer)
		filterActiveEventsButtonBuffer := new(bytes.Buffer)
		filterInactiveEventsButtonBuffer := new(bytes.Buffer)
		components.FilterAllEventsButton(pageData.ActiveDataCount+pageData.InactiveDataCount).Render(context.Background(), filterAllEventButtonBuffer)
		components.FilterActiveEventsButton(pageData.ActiveDataCount).Render(context.Background(), filterActiveEventsButtonBuffer)
		components.FilterInactiveEventsButton(pageData.InactiveDataCount).Render(context.Background(), filterInactiveEventsButtonBuffer)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           filterAllEventButtonBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           filterActiveEventsButtonBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:           filterInactiveEventsButtonBuffer.String(),
			Mode:              datastar.WithModeOuter(),
			UseViewTransition: true,
		}

	}

}
