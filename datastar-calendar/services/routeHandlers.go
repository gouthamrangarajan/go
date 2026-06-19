package services

import (
	"bytes"
	"context"
	"datastar-calendar/components"
	"datastar-calendar/components/shared"
	"datastar-calendar/models"
	"datastar-calendar/services/db"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

type monthYearDayWeek struct {
	Month time.Month
	Year  int
	Week  int
	Day   int
}

var uiSidMap sync.Map

func MonthPage(responseWriter http.ResponseWriter, request *http.Request) {
	month := request.URL.Query().Get("month")
	year := request.URL.Query().Get("year")
	model := models.MonthYearDayWeekString{Month: month, Year: year, Day: ""}
	MonthPageWithOob(responseWriter, request, model, false)
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
	uiSid := uuid.New().String()
	pageData := models.MonthPageData{
		CalendarData:        calendarData.data,
		EventsData:          eventsData,
		CurrentMonthAndYear: calendarData.monthStartDate,
		From:                from,
		UiSid:               uiSid,
	}
	components.MonthCalendarPage(pageData).Render(request.Context(), responseWriter)
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
func SSEHandler(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(TokenKey).(string)
	var signals models.ClientSignals
	datastar.ReadSignals(request, &signals)
	sessionKey := GenerateUserSessionKey(token, signals.UiSid)

	userSessionChannel := make(chan models.LongSSEData)
	uiSidMap.Store(sessionKey, userSessionChannel)

	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchSignals([]byte(`{showErrorMessage:false}`))
	for {
		select {
		case <-request.Context().Done():
			uiSidMap.Delete(sessionKey)
			return
		case channelData := <-userSessionChannel:
			if channelInMap, ok := uiSidMap.Load(sessionKey); !ok || channelInMap != userSessionChannel {
				return
			}
			switch {
			case channelData.IsRemove:
				sse.RemoveElement(channelData.Selector, datastar.WithUseViewTransitions(channelData.UseViewTransition))
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
		}
	}
}
func DetailsUI(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(TokenKey).(string)
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(request, &ClientSignals)
	key := GenerateUserSessionKey(token, ClientSignals.UiSid)

	id := chi.URLParam(request, "id")
	dataChannel := make(chan models.EventData)
	defer close(dataChannel)
	go db.GetDataById(struct {
		AccessToken string
		Id          string
	}{AccessToken: token, Id: id}, dataChannel)
	data := <-dataChannel
	dataBuffer := new(bytes.Buffer)
	components.EditEventDrawer().Render(context.Background(), dataBuffer)

	if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
		dateParsed, _ := time.Parse("2006-01-02", data.Date)
		stopAfterParsed, stopAfterErr := time.Parse("2006-01-02", data.StopAfter)
		dateFormatted := dateParsed.Format("01/02/2006")
		stopAfterFormatted := ""
		if stopAfterErr == nil {
			stopAfterFormatted = stopAfterParsed.Format("01/02/2006")
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:   fmt.Sprintf(`{eventId:"%s",task:"%s",date:"%s",frequency:"%s",stopAfter:"%s",exact:"%s"}`, data.Id, data.Task, dateFormatted, data.Frequency, stopAfterFormatted, data.Exact),
			IsSignals: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  dataBuffer.String(),
			Selector: "body",
			Mode:     datastar.WithModeAppend(),
		}
	}
}

func CloseDetailsUI(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(TokenKey).(string)
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(request, &ClientSignals)
	key := GenerateUserSessionKey(token, ClientSignals.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Selector:          "#taskDetails",
			IsRemove:          true,
			UseViewTransition: true,
		}
	}
}
func AddUI(responseWriter http.ResponseWriter, request *http.Request) {
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
	token := request.Context().Value(TokenKey).(string)
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(request, &ClientSignals)
	key := GenerateUserSessionKey(token, ClientSignals.UiSid)

	dataBuffer := new(bytes.Buffer)
	components.AddEventModal(addEventDate, week).Render(context.Background(), dataBuffer)

	if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:   fmt.Sprintf(`{eventId:'',task:'',date:'%s',frequency:'Only once',stopAfter:'',exact:''}`, addEventDate.Format("2006-01-02")),
			IsSignals: true,
		}
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content:  dataBuffer.String(),
			Selector: "body",
			Mode:     datastar.WithModeAppend(),
		}
	}
}
func CloseAddUI(responseWriter http.ResponseWriter, request *http.Request) {
	token := request.Context().Value(TokenKey).(string)
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(request, &ClientSignals)
	key := GenerateUserSessionKey(token, ClientSignals.UiSid)
	if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Selector:          "#addTask",
			IsRemove:          true,
			UseViewTransition: true,
		}
	}
}

func SaveEvent(responseWriter http.ResponseWriter, request *http.Request) {
	if strings.ToUpper(request.Method) != "POST" {
		responseWriter.WriteHeader(405)
		return
	}
	token := request.Context().Value(TokenKey).(string)
	var ClientSignals models.ClientSignals
	datastar.ReadSignals(request, &ClientSignals)
	// fmt.Println("Received signals: ", ClientSignals)
	key := GenerateUserSessionKey(token, ClientSignals.UiSid)

	dateLayout := "2006-01-02"
	if ClientSignals.EventId != "" {
		dateLayout = "01/02/2006"
	}
	stopAfterDateLayout := "01/02/2006"

	task := strings.Trim(ClientSignals.Task, "")
	date := ClientSignals.Date
	frequency := ClientSignals.Frequency
	stopAfter := ClientSignals.StopAfter
	exact := ClientSignals.Exact

	var dateParsed time.Time
	var stopAfterParsed time.Time
	var dateParseError error
	var stopAfterParseError error

	errors := []string{}

	if date == "" {
		errors = append(errors, "Date is required")
	} else {
		dateParsed, dateParseError = time.Parse(dateLayout, date)
		if dateParseError != nil {
			errors = append(errors, "Date is not in correct format")
		} else if ClientSignals.EventId != "" {
			if time.Until(dateParsed).Hours() < -24 {
				errors = append(errors, "Date should not be in the past")
			}
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
			stopAfterParsed, stopAfterParseError = time.Parse(stopAfterDateLayout, stopAfter)
			// fmt.Printf("Parsed stop after date: %v, error: %v\n", stopAfterParsed, err)
			if stopAfterParseError != nil {
				errors = append(errors, "Stop After is not in correct format")
			} else if stopAfterParsed.Sub(dateParsed).Hours() < 24 {
				errors = append(errors, "Stop After date should be after Event date")
			}
		}
	}
	if frequency == "Only once" && exact == "yes" {
		errors = append(errors, "Exact date is not allowed for only once frequency")
	}
	if len(errors) > 0 {
		dataBuffer := new(bytes.Buffer)
		dataShowAttr := "!$_addingTask"
		if ClientSignals.EventId != "" {
			dataShowAttr = "!$_savingTask"
		}
		shared.SaveEventValidationError(errors, dataShowAttr).Render(context.Background(), dataBuffer)
		if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content: dataBuffer.String(),
				Mode:    datastar.WithModeOuter(),
			}
		}
		return
	}
	var dataFromDbSave models.EventData
	parsedToken, _, tokenParseErr := jwt.NewParser().ParseUnverified(token, jwt.MapClaims{})
	var functionalityErrored bool
	if tokenParseErr != nil {
		fmt.Printf("Error parsing accesstoken %v\n", tokenParseErr)
		functionalityErrored = true
	}
	if !functionalityErrored {
		sub, subjectErr := parsedToken.Claims.GetSubject()
		if subjectErr != nil {
			fmt.Printf("Error get subject from claims %v\n", subjectErr)
			functionalityErrored = true
		}

		if subjectErr == nil {
			channel := make(chan models.EventData)
			defer close(channel)
			if stopAfterParseError == nil && stopAfter != "" {
				// fmt.Printf("Before Parsed stop after: %v\n", stopAfter)
				stopAfter = stopAfterParsed.Format("2006-01-02")
			} else {
				stopAfter = ""
			}
			// fmt.Printf("Parsed stop after: %v\n", stopAfter)
			dataToDB := models.EventData{
				Task:      task,
				Frequency: frequency,
				Date:      dateParsed.Format("2006-01-02"),
				UserId:    sub,
				Exact:     exact,
				StopAfter: stopAfter,
			}
			if ClientSignals.EventId != "" {
				dataToDB.Id = ClientSignals.EventId
				go db.UpdateData(token, dataToDB, channel)

			} else {
				go db.AddData(token, dataToDB, channel)
			}
			dataFromDbSave = <-channel
			if dataFromDbSave.Id == "" {
				functionalityErrored = true
			}
		}
	}
	dataBuffer := new(bytes.Buffer)
	if functionalityErrored {
		if ClientSignals.EventId != "" {
			components.EditEventResult(false, task).Render(context.Background(), dataBuffer)
		} else {
			components.AddEventResult(false, task).Render(context.Background(), dataBuffer)
		}
	} else {
		if ClientSignals.EventId != "" {
			components.EditEventResult(true, task).Render(context.Background(), dataBuffer)
		} else {
			components.AddEventResult(true, task).Render(context.Background(), dataBuffer)
		}
	}
	if userSession, userSessionExists := uiSidMap.Load(key); userSessionExists {
		userSession.(chan models.LongSSEData) <- models.LongSSEData{
			Content: dataBuffer.String(),
			Mode:    datastar.WithModeOuter(),
		}
		if !functionalityErrored {
			if ClientSignals.EventId == "" {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					Content:   "{eventId:'',task:'',frequency:'Only once',stopAfter:'',exact:'no'}",
					IsSignals: true,
				}
			} else {
				userSession.(chan models.LongSSEData) <- models.LongSSEData{
					IsRemove: true,
					Selector: "#event-" + ClientSignals.EventId,
				}
			}
			newEventDataUIBuffer := new(bytes.Buffer)
			components.MonthCalendarTableEventItem(dataFromDbSave).Render(context.Background(), newEventDataUIBuffer)
			userSession.(chan models.LongSSEData) <- models.LongSSEData{
				Content:  newEventDataUIBuffer.String(),
				Selector: "#eventsContainer-" + dateParsed.Format("2006-01-02"),
				Mode:     datastar.WithModeAppend(),
			}
		}
	}
}

// func UpdateDate(responseWriter http.ResponseWriter, request *http.Request) {
// 	var dnd models.DnD
// 	jsonErr := json.NewDecoder(request.Body).Decode(&dnd)
// 	if jsonErr != nil {
// 		responseWriter.WriteHeader(400)
// 		return
// 	}
// 	token := request.Context().Value(TokenKey).(string)
// 	channel := make(chan bool)
// 	go db.UpdateDate(token, dnd, channel)
// 	ret := <-channel

// 	if ret {
// 		responseWriter.Write([]byte("Success"))
// 		return
// 	}
// 	responseWriter.WriteHeader(500)
// }

// func AddPageWithOob(responseWriter http.ResponseWriter, request *http.Request, from models.MonthYearDayWeekString, isOob bool) {
// 	today := time.Now()
// 	year := today.Year()
// 	month := today.Month()
// 	day := today.Day()
// 	week := 0
// 	fromWeek := request.URL.Query().Get("week")
// 	token := request.Context().Value(TokenKey).(string)
// 	if from.Month != "" {
// 		monthFromUrl, err := strconv.Atoi(from.Month)
// 		if err == nil {
// 			month = time.Month(monthFromUrl)
// 		}
// 	}
// 	if from.Year != "" {
// 		yearFromUrl, err := strconv.Atoi(from.Year)
// 		if err == nil {
// 			year = yearFromUrl
// 		}
// 	}
// 	if from.Day != "" {
// 		dayFromUrl, err := strconv.Atoi(from.Day)
// 		if err == nil {
// 			day = dayFromUrl
// 		}
// 	}
// 	if fromWeek != "" {
// 		weekFromUrl, err := strconv.Atoi(fromWeek)
// 		if err == nil {
// 			week = weekFromUrl
// 		}
// 	}
// 	var calendarData calendarDataType
// 	if week == 0 {
// 		calendarData = generateCalendarData(year, month, today.Location())
// 	} else {
// 		calendarData = generateWeekCalendarData(monthYearDayWeek{Year: year, Month: month, Week: week}, today.Location())
// 	}
// 	channel := make(chan []models.EventData)
// 	go db.GetData(token, calendarData.calendarDaysStrFormat, channel)
// 	eventsData := <-channel
// 	addEventDate := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
// 	if week == 0 {
// 		components.AddEventPage(calendarData.data, eventsData, addEventDate, isOob).Render(request.Context(), responseWriter)
// 	} else {
// 		components.AddEventPageWeek(calendarData.data, eventsData, addEventDate, week, isOob).Render(request.Context(), responseWriter)
// 	}
// }

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
