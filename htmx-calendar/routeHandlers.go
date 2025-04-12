package main

import (
	"encoding/json"
	"fmt"
	"htmx-calendar/components"
	"htmx-calendar/models"
	"htmx-calendar/services"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type calendarDataType struct {
	data                  [][7]time.Time
	monthStartDate        time.Time
	monthEndDate          time.Time
	calendarStartDate     time.Time
	calendarEndDate       time.Time
	calendarDaysStrFormat []string
}

func MonthPage(responseWriter http.ResponseWriter, request *http.Request, token string) {
	month := request.URL.Query().Get("month")
	year := request.URL.Query().Get("year")
	MonthPageWithOob(responseWriter, request, token, month, year, "", false)
}
func MonthPageWithOob(responseWriter http.ResponseWriter, request *http.Request, token string, toMonth string, toYear string, toDay string, isOob bool) {
	from := request.URL.Query().Get("from")
	today := time.Now()
	year := today.Year()
	month := today.Month()
	if toMonth != "" {
		monthFromUrl, err := strconv.Atoi(toMonth)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if toYear != "" {
		yearFromUrl, err := strconv.Atoi(toYear)
		if err == nil {
			year = yearFromUrl
		}
	}
	calendarData := generateCalendarData(year, month, today.Location())
	channel := make(chan []models.EventData)
	go services.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	components.MonthCalendarPage(calendarData.data, eventsData, calendarData.monthStartDate, from, isOob).Render(request.Context(), responseWriter)
}
func generateCalendarData(year int, month time.Month, location *time.Location) calendarDataType {
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
	allDatesToFilter := generateAllDatesStringFromStartToEnd(startDateForCalendar, endDateForCalendar)
	ret.data = data
	ret.calendarStartDate = startDateForCalendar
	ret.calendarEndDate = endDateForCalendar
	ret.monthStartDate = startDateOfMonth
	ret.monthEndDate = endDateOfMonth
	ret.calendarDaysStrFormat = allDatesToFilter
	return ret
}
func generateAllDatesStringFromStartToEnd(start time.Time, end time.Time) []string {
	ret := []string{}
	loopDt := start
	ret = append(ret, loopDt.Format("2006-01-02"))
	for {
		loopDt = loopDt.AddDate(0, 0, 1)
		if end.Sub(loopDt) <= 0 {
			break
		}
		ret = append(ret, loopDt.Format("2006-01-02"))
	}
	return ret
}

func UpdateDate(responseWriter http.ResponseWriter, request *http.Request, token string) {
	var dnd models.DnD
	jsonErr := json.NewDecoder(request.Body).Decode(&dnd)
	if jsonErr != nil {
		responseWriter.WriteHeader(400)
		return
	}
	channel := make(chan bool)
	go services.UpdateDate(token, dnd.Id, dnd.Date, channel)
	ret := <-channel

	if ret {
		responseWriter.Write([]byte("Success"))
		return
	}
	responseWriter.WriteHeader(500)
}

func AddPage(responseWriter http.ResponseWriter, request *http.Request, token string) {
	if strings.ToUpper(request.Method) == "GET" {
		fromMonth := request.URL.Query().Get("month")
		fromYear := request.URL.Query().Get("year")
		fromDay := request.URL.Query().Get("day")
		AddPageWithOob(responseWriter, request, token, fromMonth, fromYear, fromDay, false)
	} else if strings.ToUpper(request.Method) == "POST" {
		task := request.FormValue("task")
		task = strings.Trim(task, "")
		date := request.FormValue("date")
		frequency := request.FormValue("frequency")
		parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
		if err != nil {
			fmt.Printf("Error parsing accesstoken %v\n", err)
		}
		if err == nil && len(task) > 3 {
			sub, err := parsedToken.Claims.GetSubject()
			if err != nil {
				fmt.Printf("Error get subject from claims %v\n", err)
			} else {
				channel := make(chan int16)
				go services.AddData(token, models.EventData{
					Task:      task,
					Frequency: frequency,
					Date:      date,
					UserId:    sub,
				}, channel)
				rowsAffected := <-channel
				if rowsAffected > 0 {
					components.AddEventResult(true, task).Render(request.Context(), responseWriter)
					return
				}
			}
		}
		components.AddEventResult(false, task).Render(request.Context(), responseWriter)
	} else {
		responseWriter.WriteHeader(405)
	}
}

func AddPageWithOob(responseWriter http.ResponseWriter, request *http.Request, token string, fromMonth string, fromYear string, fromDay string, isOob bool) {
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
	var calendarData calendarDataType
	if week == 0 {
		calendarData = generateCalendarData(year, month, today.Location())
	} else {
		calendarData = generateWeekCalendarData(year, month, week, today.Location())
	}
	channel := make(chan []models.EventData)
	go services.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	addEventDate := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
	if week == 0 {
		components.AddEventPage(calendarData.data, eventsData, addEventDate, isOob).Render(request.Context(), responseWriter)
	} else {
		components.AddEventPageWeek(calendarData.data, eventsData, addEventDate, week, isOob).Render(request.Context(), responseWriter)
	}
}

func WeekPage(responseWriter http.ResponseWriter, request *http.Request, token string) {
	toMonth := request.URL.Query().Get("month")
	toYear := request.URL.Query().Get("year")
	toWeek := request.URL.Query().Get("week")
	WeekPageWithOob(responseWriter, request, token, toMonth, toYear, toWeek, false)
}

func WeekPageWithOob(responseWriter http.ResponseWriter, request *http.Request, token string, toMonth string, toYear string, toWeek string, isOob bool) {
	today := time.Now()
	year := today.Year()
	month := today.Month()
	week := 1
	from := request.URL.Query().Get("from")

	if toMonth != "" {
		monthFromUrl, err := strconv.Atoi(toMonth)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if toYear != "" {
		yearFromUrl, err := strconv.Atoi(toYear)
		if err == nil {
			year = yearFromUrl
		}
	}
	if toWeek != "" {
		weekFromUrl, err := strconv.Atoi(toWeek)
		if err == nil {
			week = weekFromUrl
		}
	}
	calendarData := generateWeekCalendarData(year, month, week, today.Location())
	channel := make(chan []models.EventData)
	go services.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	components.WeekCalendarPage(calendarData.data, eventsData, calendarData.monthStartDate, from, week, isOob).Render(request.Context(), responseWriter)
}

func generateWeekCalendarData(year int, month time.Month, week int, location *time.Location) calendarDataType {
	ret := calendarDataType{}
	startDateOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, location)
	startDateForMonthCalendar := startDateOfMonth.AddDate(0, 0, -int(startDateOfMonth.Weekday()))
	endDateOfMonth := time.Date(year, month+1, 0, 23, 59, 0, 0, location)

	startDateForWeek := startDateForMonthCalendar.AddDate(0, 0, int(week-1)*7)

	data := make([][7]time.Time, 1)

	for idx := range 7 {
		data[0][idx] = startDateForWeek.AddDate(0, 0, idx)
	}
	allDatesToFilter := generateAllDatesStringFromStartToEnd(data[0][0], data[0][6])
	ret.calendarDaysStrFormat = allDatesToFilter
	ret.monthStartDate = startDateOfMonth
	ret.monthEndDate = endDateOfMonth
	ret.calendarStartDate = startDateForWeek
	ret.calendarEndDate = data[0][6]
	ret.data = data
	return ret

}

func DeleteEvent(responseWriter http.ResponseWriter, request *http.Request, token string) {
	if strings.ToUpper(request.Method) != "DELETE" {
		responseWriter.WriteHeader(405)
	} else {
		eventId := request.FormValue("eventId")
		if eventId == "" {
			responseWriter.WriteHeader(400)
			return
		}
		channel := make(chan bool)
		go services.DeleteEvent(token, eventId, channel)
		ret := <-channel

		if ret {
			responseWriter.WriteHeader(200)
			return
		}
		responseWriter.WriteHeader(500)
	}
}
