package handlers

import (
	"bytes"
	"context"
	"datastar-calendar/internal/models"
	"datastar-calendar/internal/views/pages"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type calendarDataType struct {
	data                  [][7]time.Time
	monthStartDate        time.Time
	monthEndDate          time.Time
	calendarStartDate     time.Time
	calendarEndDate       time.Time
	calendarDaysStrFormat []string
}
type MonthHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewMonthHandler(dbService *db.DBService, uidSidMap *sync.Map) *MonthHandler {
	return &MonthHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (m *MonthHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	month := request.URL.Query().Get("month")
	year := request.URL.Query().Get("year")
	model := models.MonthYearDayWeekString{Month: month, Year: year, Day: ""}
	m.monthPageWithData(responseWriter, request, model)
}
func (m *MonthHandler) monthPageWithData(responseWriter http.ResponseWriter, request *http.Request, to models.MonthYearDayWeekString) {
	token := request.Context().Value(services.TokenKey).(string)
	from := request.URL.Query().Get("from")
	today := time.Now()
	year := today.Year()
	month := today.Month()
	if to.Month != "" {
		monthFromUrl, err := strconv.Atoi(to.Month)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if to.Year != "" {
		yearFromUrl, err := strconv.Atoi(to.Year)
		if err == nil {
			year = yearFromUrl
		}
	}

	calendarData := m.generateCalendarData(year, month, today.Location())
	channel := make(chan []models.EventData)
	go m.dbService.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	pageData := models.MonthCalendarData{
		CalendarData:        calendarData.data,
		EventsData:          eventsData,
		CurrentMonthAndYear: calendarData.monthStartDate,
		From:                from,
	}
	if request.Header.Get("Datastar-Request") == "true" {
		var clientSignals models.ClientSignals
		datastar.ReadSignals(request, &clientSignals)
		key := services.GenerateUserSessionKey(token, clientSignals.UiSid)
		dataBuffer := new(bytes.Buffer)
		pages.MonthCalendar(pageData).Render(context.Background(), dataBuffer)
		if userSession, userSessionExists := m.uiSidMap.Load(key); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           dataBuffer.String(),
				Mode:              datastar.WithModeOuter(),
				UseViewTransition: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `window.history.replaceState({}, "", "?month=` + strconv.Itoa(int(month)) + `&year=` + strconv.Itoa(year) + `");`,
				IsScript: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:   `{pageLoading:false}`,
				IsSignals: true,
			}
		}
		return
	}
	pages.MonthCalendarPage(pageData).Render(request.Context(), responseWriter)
}
func (m *MonthHandler) generateCalendarData(year int, month time.Month, location *time.Location) calendarDataType {
	ret := calendarDataType{}
	startDateOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, location)
	startDateForCalendar := startDateOfMonth.AddDate(0, 0, -int(startDateOfMonth.Weekday()))
	endDateOfMonth := time.Date(year, month+1, 0, 23, 59, 0, 0, location)
	endDateForCalendar := endDateOfMonth.AddDate(0, 0, 6-int(endDateOfMonth.Weekday()))
	numberOfDays := math.Round(endDateForCalendar.Sub(startDateForCalendar).Hours() / 24)
	rows := int(numberOfDays / 7)
	data := make([][7]time.Time, rows)
	number := 0
	for row := range rows {
		for col := range 7 {
			data[row][col] = startDateForCalendar.AddDate(0, 0, number)
			number++
		}
	}
	allDatesToFilter := services.GenerateAllDatesStringFromStartToEnd(startDateForCalendar, endDateForCalendar)
	ret.data = data
	ret.calendarStartDate = startDateForCalendar
	ret.calendarEndDate = endDateForCalendar
	ret.monthStartDate = startDateOfMonth
	ret.monthEndDate = endDateOfMonth
	ret.calendarDaysStrFormat = allDatesToFilter
	return ret
}
