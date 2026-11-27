package models

import "time"

type GoToDay struct {
	GoToMonth int
	GoToYear  int
	GoToDay   int
	CurrMonth int
	CurrYear  int
	CurrDay   int
}

type TableTdTemplateData struct {
	Row     int
	Col     int
	From    string
	DaysLen int
	DataLen int
}

type TableTdData struct {
	Data                time.Time
	CurrentMonthAndYear time.Time
}
type MonthCalendarData struct {
	CalendarData        [][7]time.Time
	EventsData          []EventData
	CurrentMonthAndYear time.Time
	From                string
}
type WeekCalendarData struct {
	CalendarData        [][7]time.Time
	EventsData          []EventData
	CurrentMonthAndYear time.Time
	From                string
	Week                int
}
type ClientSignals struct {
	UiSid       string `json:"uiSid"`
	EventId     string `json:"eventId"`
	Task        string `json:"task"`
	Frequency   string `json:"frequency"`
	StopAfter   string `json:"stopAfter"`
	Date        string `json:"date"`
	DateChanged bool   `json:"dateChanged"`
	Exact       string `json:"exact"`
}
