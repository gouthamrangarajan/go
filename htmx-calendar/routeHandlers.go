package main

import (
	"htmx-calendar/components"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/a-h/templ"
)

func MainPage(responseWriter http.ResponseWriter, request *http.Request) {
	nextMonth := request.URL.Query().Get("month")
	nextYear := request.URL.Query().Get("year")
	from := request.URL.Query().Get("from")

	today := time.Now()
	year := today.Year()
	month := today.Month()

	if nextMonth != "" {
		monthFromUrl, err := strconv.Atoi(nextMonth)
		if err == nil {
			month = time.Month(monthFromUrl)
		}
	}
	if nextYear != "" {
		yearFromUrl, err := strconv.Atoi(nextYear)
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

	data := make([][7]time.Time, rows)
	number := 0
	for row := range rows {
		for col := range 7 {
			data[row][col] = startDateForCalendar.AddDate(0, 0, number)
			number++
		}
	}
	component := components.Main(data, startDate, from)
	templ.Handler(component).Component.Render(request.Context(), responseWriter)
}
