package models

type MainElData struct {
	Location    string
	Sort        string
	Suggestions string
	SId         string
}

type ClientSignals struct {
	SId string `json:"sId"`
}
