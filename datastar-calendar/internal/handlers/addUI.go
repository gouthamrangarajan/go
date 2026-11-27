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
	"strconv"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type AddUIHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewAddUIHandler(dbService *db.DBService, uidSidMap *sync.Map) *AddUIHandler {
	return &AddUIHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (a *AddUIHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	fromMonth := request.URL.Query().Get("month")
	fromYear := request.URL.Query().Get("year")
	fromDay := request.URL.Query().Get("day")
	today := time.Now()
	year := today.Year()
	month := today.Month()
	day := today.Day()
	week := 0
	fromWeek := request.URL.Query().Get("week")
	if fromMonth != "" {
		monthFromUrl, err := strconv.Atoi(fromMonth)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if fromYear != "" {
		yearFromUrl, err := strconv.Atoi(fromYear)
		if err == nil {
			year = yearFromUrl
		}
	}
	if fromDay != "" {
		dayFromUrl, err := strconv.Atoi(fromDay)
		if err == nil {
			day = dayFromUrl
		}
	}
	if fromWeek != "" {
		weekFromUrl, err := strconv.Atoi(fromWeek)
		if err == nil {
			week = weekFromUrl
		}
	}
	addEventDate := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)

	dataBuffer := new(bytes.Buffer)
	components.AddEventModal(addEventDate, week).Render(context.Background(), dataBuffer)

	if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:   fmt.Sprintf(`{pageLoading:false,eventId:'',task:'',date:'%s',frequency:'Only once',stopAfter:'',exact:''}`, addEventDate.Format("2006-01-02")),
			IsSignals: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  dataBuffer.String(),
			Selector: "body",
			Mode:     datastar.WithModeAppend(),
		}
	}
}
func (a *AddUIHandler) Close(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(services.TokenKey).(string)
	var clientSignals models.ClientSignals
	datastar.ReadSignals(request, &clientSignals)
	key := services.GenerateUserSessionKey(token, clientSignals.UiSid)
	if userSession, userSessionExists := a.uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Selector:          "#addTask",
			IsRemove:          true,
			UseViewTransition: true,
		}
	}
}
