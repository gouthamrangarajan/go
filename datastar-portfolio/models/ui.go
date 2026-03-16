package models

type ClientSignals struct {
	Id            string   `json:"id"`
	ServiceFilter []string `json:"serviceFilter"`
	SrchTxt       string   `json:"srchTxt"`
	Email         string   `json:"email"`
	Message       string   `json:"message"`
}
