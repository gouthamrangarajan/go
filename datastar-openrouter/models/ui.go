package models

type ClientSignals struct {
	Prompt    string `json:"prompt"`
	SessionId int    `json:"sessionId"`
}

type UICookie struct {
	Name  string
	Value string
}
