package main

import (
	"encoding/json"
	"htmx-calendar/components"
	"htmx-calendar/models"
	"htmx-calendar/services"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

func MainOrLoginPage(responseWriter http.ResponseWriter, request *http.Request) {
	month := request.URL.Query().Get("month")
	year := request.URL.Query().Get("year")
	// cookie check
	cookie, err := request.Cookie("token")
	if err != nil {
		components.LoginPage(true, month, year).Render(request.Context(), responseWriter)
	} else {
		token := cookie.Value
		if token == "" {
			components.LoginPage(true, month, year).Render(request.Context(), responseWriter)
		} else {
			mainPage(responseWriter, request, token, month, year, false)
		}
	}
}

func Login(responseWriter http.ResponseWriter, request *http.Request) {
	email := request.FormValue("email")
	password := request.FormValue("password")
	channel := make(chan services.LoginResponse)
	defer close(channel)
	go services.Login(services.LoginRequest{Email: email, Password: password}, channel)
	resp := <-channel
	if resp.ErrorCode != "" {
		// fmt.Printf("%v\n", time.Now().Add(86400*time.Second))
		components.LoginError().Render(request.Context(), responseWriter)
	} else {
		secure := true
		if os.Getenv("Env") == "Development" {
			secure = false
		}
		cookie := http.Cookie{
			Name:     "token",
			Value:    resp.AccessToken,
			Expires:  time.Now().Add(time.Duration(resp.ExpiresIn-120) * time.Second),
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(responseWriter, &cookie)
		month := request.FormValue("month")
		year := request.FormValue("year")
		mainPage(responseWriter, request, resp.AccessToken, month, year, true)
	}
}

func mainPage(responseWriter http.ResponseWriter, request *http.Request, accessToken string, toMonth string, toYear string, isOob bool) {
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
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, today.Location())
	startDateForCalendar := startDate.AddDate(0, 0, -int(startDate.Weekday()))
	endDate := time.Date(year, month+1, 0, 23, 59, 0, 0, today.Location())
	endDateForCalendar := endDate.AddDate(0, 0, 6-int(endDate.Weekday()))
	numberOfDays := math.Round(endDateForCalendar.Sub(startDateForCalendar).Hours() / 24)
	rows := int(numberOfDays / 7)
	calendarData := make([][7]time.Time, rows)
	number := 0
	for row := range rows {
		for col := range 7 {
			calendarData[row][col] = startDateForCalendar.AddDate(0, 0, number)
			number++
		}
	}
	allDatesToFilter := generateAllDatesStringFromStartToEnd(startDateForCalendar, endDateForCalendar)
	channel := make(chan []models.CalendarData)
	go services.GetData(accessToken, allDatesToFilter, channel)
	eventsData := <-channel
	if isOob {
		components.MainPageWithoutLayout(calendarData, eventsData, startDate, from, true).Render(request.Context(), responseWriter)
	} else {
		components.MainPage(calendarData, eventsData, startDate, from).Render(request.Context(), responseWriter)
	}
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

func UpdateDate(responseWriter http.ResponseWriter, request *http.Request) {
	//check cookies & send unauthorized
	cookie, err := request.Cookie("token")
	token := ""
	if err == nil {
		token = cookie.Value
	}
	if err != nil || token == "" {
		responseWriter.WriteHeader(401)
		responseWriter.Write([]byte("UnAuthorized"))
		return
	}
	var dnd models.DnD
	jsonErr := json.NewDecoder(request.Body).Decode(&dnd)
	if jsonErr != nil {
		responseWriter.WriteHeader(400)
		responseWriter.Write([]byte("Bad Request"))
		return
	}
	channel := make(chan bool)
	go services.UpdateDate(token, dnd.Id, dnd.Date, channel)
	ret := <-channel

	if ret {
		responseWriter.WriteHeader(200)
		responseWriter.Write([]byte("Success"))
		return
	}
	responseWriter.WriteHeader(500)
	responseWriter.Write([]byte("Internal Server Error"))
}
