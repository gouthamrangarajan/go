package models

type ClientSignals struct {
	ServiceFilter []string `json:"serviceFilter"`
	SrchTxt       string   `json:"srchTxt"`
	Email         string   `json:"email"`
	Message       string   `json:"message"`
}
