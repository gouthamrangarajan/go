package models

type EventData struct {
	Id        string `json:"id"`
	Task      string `json:"task"`
	Frequency string `json:"frequency"`
	Date      string `json:"date"`
	UserId    string `json:"user_id"`
	StopAfter string `json:"stopAfter"`
	Exact     string `json:"exact"`
}
