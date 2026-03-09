package models

type ClientSignals struct {
	ServiceFilter []string `json:"serviceFilter"`
	SrchTxt       string   `json:"srchTxt"`
}
