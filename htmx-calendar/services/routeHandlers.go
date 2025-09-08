package services

import (
	"encoding/json"
	"fmt"
	"htmx-calendar/components"
	"htmx-calendar/models"
	"htmx-calendar/services/db"
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

type monthYearDayWeek struct {
	Month time.Month
	Year  int
	Week  int
	Day   int
}

func MonthPage(responseWriter http.ResponseWriter, request *http.Request) {
	month := request.URL.Query().Get("month")
	year := request.URL.Query().Get("year")
	model := models.MonthYearDayWeekString{Month: month, Year: year, Day: ""}
	if request.Header.Get("HX-Request") == "true" {
		MonthPageWithOob(responseWriter, request, model, true)
	} else {
		MonthPageWithOob(responseWriter, request, model, false)
	}
}
func MonthPageWithOob(responseWriter http.ResponseWriter, request *http.Request, to models.MonthYearDayWeekString, isOob bool) {
	token := request.Context().Value(TokenKey).(string)
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
	calendarData := generateCalendarData(year, month, today.Location())
	channel := make(chan []models.EventData)
	go db.GetData(token, calendarData.calendarDaysStrFormat, channel)
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
		if end.Sub(loopDt) < 0 {
			break
		}
		ret = append(ret, loopDt.Format("2006-01-02"))
	}
	return ret
}

func UpdateDate(responseWriter http.ResponseWriter, request *http.Request) {
	var dnd models.DnD
	jsonErr := json.NewDecoder(request.Body).Decode(&dnd)
	if jsonErr != nil {
		responseWriter.WriteHeader(400)
		return
	}
	token := request.Context().Value(TokenKey).(string)
	channel := make(chan bool)
	go db.UpdateDate(token, dnd, channel)
	ret := <-channel

	if ret {
		responseWriter.Write([]byte("Success"))
		return
	}
	responseWriter.WriteHeader(500)
}

func AddPage(responseWriter http.ResponseWriter, request *http.Request) {
	if strings.ToUpper(request.Method) == "GET" {
		fromMonth := request.URL.Query().Get("month")
		fromYear := request.URL.Query().Get("year")
		fromDay := request.URL.Query().Get("day")
		model := models.MonthYearDayWeekString{Month: fromMonth, Year: fromYear, Day: fromDay}
		if request.Header.Get("HX-Request") == "true" {
			AddPageWithOob(responseWriter, request, model, true)
		} else {
			AddPageWithOob(responseWriter, request, model, false)
		}
	} else if strings.ToUpper(request.Method) == "POST" {
		dateLayout := "2006-01-02"
		token := request.Context().Value(TokenKey).(string)
		task := request.FormValue("task")
		task = strings.Trim(task, "")
		date := request.FormValue("date")
		var dateFormatted time.Time
		frequency := request.FormValue("frequency")
		stopAfter := request.FormValue("stopAfter")
		var stopAfterFormatted time.Time
		exact := request.FormValue("exact")
		errors := []string{}
		parsedToken, _, err := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
		if err != nil {
			fmt.Printf("Error parsing accesstoken %v\n", err)
		}
		if date == "" {
			errors = append(errors, "Date is required")
		} else {
			dateFormatted, err = time.Parse(dateLayout, date)
			if err != nil {
				errors = append(errors, "Date is not in correct format")
			}
		}
		if len(task) <= 3 {
			errors = append(errors, "Task should be more than 3 characters")
		}
		frequencyAllowed := false
		for _, frequencyFromArray := range AllowedFrequencies {
			if frequencyFromArray == frequency {
				frequencyAllowed = true
				break
			}
		}
		if !frequencyAllowed {
			errors = append(errors, "Frequency is not allowed")
		} else if frequency == "Only once" && stopAfter != "" {
			errors = append(errors, "Stop after is not allowed for only once frequency")
		} else if frequency != "Only once" {
			if stopAfter != "" {
				stopAfterFormatted, err = time.Parse(dateLayout, stopAfter)
				if err != nil {
					errors = append(errors, "Stop After is not in correct format")
				} else if stopAfterFormatted.Sub(dateFormatted).Hours() < 24 {
					errors = append(errors, "Stop After date should be after Event date")
				}
			}
		}
		if frequency == "Only once" && exact == "yes" {
			errors = append(errors, "Exact date is not allowed for only once frequency")
		}

		if len(errors) > 0 {
			components.AddEventValidationError(errors).Render(request.Context(), responseWriter)
			return
		}
		if err == nil {
			sub, err := parsedToken.Claims.GetSubject()
			if err != nil {
				fmt.Printf("Error get subject from claims %v\n", err)
			} else {
				channel := make(chan int16)
				go db.AddData(token, models.EventData{
					Task:      task,
					Frequency: frequency,
					Date:      date,
					UserId:    sub,
					Exact:     exact,
					StopAfter: stopAfter,
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

func AddPageWithOob(responseWriter http.ResponseWriter, request *http.Request, from models.MonthYearDayWeekString, isOob bool) {
	today := time.Now()
	year := today.Year()
	month := today.Month()
	day := today.Day()
	week := 0
	fromWeek := request.URL.Query().Get("week")
	token := request.Context().Value(TokenKey).(string)
	if from.Month != "" {
		monthFromUrl, err := strconv.Atoi(from.Month)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if from.Year != "" {
		yearFromUrl, err := strconv.Atoi(from.Year)
		if err == nil {
			year = yearFromUrl
		}
	}
	if from.Day != "" {
		dayFromUrl, err := strconv.Atoi(from.Day)
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
		calendarData = generateWeekCalendarData(monthYearDayWeek{Year: year, Month: month, Week: week}, today.Location())
	}
	channel := make(chan []models.EventData)
	go db.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	addEventDate := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
	if week == 0 {
		components.AddEventPage(calendarData.data, eventsData, addEventDate, isOob).Render(request.Context(), responseWriter)
	} else {
		components.AddEventPageWeek(calendarData.data, eventsData, addEventDate, week, isOob).Render(request.Context(), responseWriter)
	}
}

func WeekPage(responseWriter http.ResponseWriter, request *http.Request) {
	toMonth := request.URL.Query().Get("month")
	toYear := request.URL.Query().Get("year")
	toWeek := request.URL.Query().Get("week")
	model := models.MonthYearDayWeekString{Month: toMonth, Year: toYear, Week: toWeek}
	if request.Header.Get("HX-Request") == "true" {
		WeekPageWithOob(responseWriter, request, model, true)
	} else {
		WeekPageWithOob(responseWriter, request, model, false)
	}
}

func WeekPageWithOob(responseWriter http.ResponseWriter, request *http.Request, to models.MonthYearDayWeekString, isOob bool) {
	today := time.Now()
	year := today.Year()
	month := today.Month()
	week := 1
	from := request.URL.Query().Get("from")
	token := request.Context().Value(TokenKey).(string)

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
	calendarData := generateWeekCalendarData(monthYearDayWeek{Year: year, Month: month, Week: week}, today.Location())
	channel := make(chan []models.EventData)
	go db.GetData(token, calendarData.calendarDaysStrFormat, channel)
	eventsData := <-channel
	components.WeekCalendarPage(calendarData.data, eventsData, calendarData.monthStartDate, from, week, isOob).Render(request.Context(), responseWriter)
}

func generateWeekCalendarData(model monthYearDayWeek, location *time.Location) calendarDataType {
	ret := calendarDataType{}
	startDateOfMonth := time.Date(model.Year, model.Month, 1, 0, 0, 0, 0, location)
	startDateForMonthCalendar := startDateOfMonth.AddDate(0, 0, -int(startDateOfMonth.Weekday()))
	endDateOfMonth := time.Date(model.Year, model.Month+1, 0, 23, 59, 0, 0, location)

	startDateForWeek := startDateForMonthCalendar.AddDate(0, 0, int(model.Week-1)*7)

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

func DeleteEvent(responseWriter http.ResponseWriter, request *http.Request) {
	eventId := request.FormValue("eventId")
	if eventId == "" {
		responseWriter.WriteHeader(400)
		return
	}
	token := request.Context().Value(TokenKey).(string)
	channel := make(chan bool)
	go db.DeleteEvent(models.DeleteEvent{AccessToken: token, Id: eventId}, channel)
	ret := <-channel

	if ret {
		responseWriter.WriteHeader(200)
		return
	}
	responseWriter.WriteHeader(500)
}
