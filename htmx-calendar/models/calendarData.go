package models

type CalendarData struct {
	Id        string `json:"id"`
	Task      string `json:"task"`
	Frequency string `json:"frequency"`
	Date      string `json:"date"`
}
