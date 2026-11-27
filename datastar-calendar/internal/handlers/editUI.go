package handlers

import (
	"bytes"
	"context"
	"datastar-calendar/internal/models"
	"datastar-calendar/internal/views/components"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

type EditUIHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewEditUIHandler(dbService *db.DBService, uidSidMap *sync.Map) *EditUIHandler {
	return &EditUIHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (e *EditUIHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)

	id := chi.URLParam(request, "id")
	dataChannel := make(chan models.EventData)
	go e.dbService.GetDataById(struct {
		AccessToken string
		Id          string
	}{AccessToken: token, Id: id}, dataChannel)
	data := <-dataChannel
	dataBuffer := new(bytes.Buffer)
	components.EditEventDrawer().Render(context.Background(), dataBuffer)

	if userSession, userSessionExists := e.uiSidMap.Load(key); userSessionExists {
		dateParsed, _ := time.Parse("2006-01-02", data.Date)
		stopAfterParsed, stopAfterErr := time.Parse("2006-01-02", data.StopAfter)
		dateFormatted := dateParsed.Format("01/02/2006")
		stopAfterFormatted := ""
		if stopAfterErr == nil {
			stopAfterFormatted = stopAfterParsed.Format("01/02/2006")
		}
		// fmt.Printf(`{pageLoading:false,eventId:"%s",task:"%s",date:"%s",dateChanged:false,frequency:"%s",stopAfter:"%s",exact:"%s"}\n`, data.Id, data.Task, dateFormatted, data.Frequency, stopAfterFormatted, data.Exact)
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:   fmt.Sprintf(`{pageLoading:false,eventId:"%s",task:"%s",date:"%s",dateChanged:false,frequency:"%s",stopAfter:"%s",exact:"%s"}`, data.Id, data.Task, dateFormatted, data.Frequency, stopAfterFormatted, data.Exact),
			IsSignals: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  dataBuffer.String(),
			Selector: "body",
			Mode:     datastar.WithModeAppend(),
		}
	}
}

func (e *EditUIHandler) Close(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)
	if userSession, userSessionExists := e.uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Selector:          "#taskDetails",
			IsRemove:          true,
			UseViewTransition: true,
		}
	}
}
