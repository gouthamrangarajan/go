package models

type EventData struct {
	Id        string `json:"id"`
	Task      string `json:"task"`
	Frequency string `json:"frequency"`
	Date      string `json:"date"`
}
