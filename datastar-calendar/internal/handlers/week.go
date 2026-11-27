package handlers

import (
	"bytes"
	"context"
	"datastar-calendar/internal/models"
	"datastar-calendar/internal/views/pages"
	"datastar-calendar/services"
	"datastar-calendar/services/db"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type monthYearDayWeek struct {
	Month time.Month
	Year  int
	Week  int
	Day   int
}

type WeekHandler struct {
	dbService *db.DBService
	uiSidMap  *sync.Map
}

func NewWeekHandler(dbService *db.DBService, uidSidMap *sync.Map) *WeekHandler {
	return &WeekHandler{
		dbService: dbService,
		uiSidMap:  uidSidMap,
	}
}

func (w *WeekHandler) Handle(responseWriter http.ResponseWriter, request *http.Request) {
	toMonth := request.URL.Query().Get("month")
	toYear := request.URL.Query().Get("year")
	toWeek := request.URL.Query().Get("week")
	model := models.MonthYearDayWeekString{Month: toMonth, Year: toYear, Week: toWeek}
	w.weekPageWithData(responseWriter, request, model)
}

func (w *WeekHandler) weekPageWithData(responseWriter http.ResponseWriter, request *http.Request, to models.MonthYearDayWeekString) {
	today := time.Now()
	year := today.Year()
	month := today.Month()
	week := 1
	from := request.URL.Query().Get("from")
	token := request.Context().Value(services.TokenKey).(string)

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
	if to.Week != "" {
		weekFromUrl, err := strconv.Atoi(to.Week)
		if err == nil {
			week = weekFromUrl
		}
	}
	calendarData := w.generateWeekCalendarData(monthYearDayWeek{Year: year, Month: month, Week: week}, today.Location())
	channel := make(chan []models.EventData)
	go w.dbService.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	pageData := models.WeekCalendarData{
		EventsData:          eventsData,
		CalendarData:        calendarData.data,
		From:                from,
		Week:                week,
		CurrentMonthAndYear: calendarData.monthStartDate,
	}
	if request.Header.Get("Datastar-Request") == "true" {
		var clientSignals models.ClientSignals
		datastar.ReadSignals(request, &clientSignals)
		key := services.GenerateUserSessionKey(token, clientSignals.UiSid)
		dataBuffer := new(bytes.Buffer)
		pages.WeekCalendar(pageData).Render(context.Background(), dataBuffer)
		if userSession, userSessionExists := w.uiSidMap.Load(key); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:           dataBuffer.String(),
				Mode:              datastar.WithModeOuter(),
				UseViewTransition: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  `window.history.replaceState({}, "", "/wk?month=` + strconv.Itoa(int(month)) + `&year=` + strconv.Itoa(year) + `&week=` + strconv.Itoa(week) + `");`,
				IsScript: true,
			}
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:   `{pageLoading:false}`,
				IsSignals: true,
			}
		}
		return
	}
	pages.WeekCalendarPage(pageData).Render(request.Context(), responseWriter)
}

func (w *WeekHandler) generateWeekCalendarData(model monthYearDayWeek, location *time.Location) calendarDataType {
	ret := calendarDataType{}
	startDateOfMonth := time.Date(model.Year, model.Month, 1, 0, 0, 0, 0, location)
	startDateForMonthCalendar := startDateOfMonth.AddDate(0, 0, -int(startDateOfMonth.Weekday()))
	endDateOfMonth := time.Date(model.Year, model.Month+1, 0, 23, 59, 0, 0, location)

	startDateForWeek := startDateForMonthCalendar.AddDate(0, 0, int(model.Week-1)*7)

	data := make([][7]time.Time, 1)

	for idx := range 7 {
		data[0][idx] = startDateForWeek.AddDate(0, 0, idx)
	}
	allDatesToFilter := services.GenerateAllDatesStringFromStartToEnd(data[0][0], data[0][6])
	ret.calendarDaysStrFormat = allDatesToFilter
	ret.monthStartDate = startDateOfMonth
	ret.monthEndDate = endDateOfMonth
	ret.calendarStartDate = startDateForWeek
	ret.calendarEndDate = data[0][6]
	ret.data = data
	return ret

}
