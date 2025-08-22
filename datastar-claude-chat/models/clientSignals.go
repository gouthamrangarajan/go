package models

type ClientSignals struct {
	SessionId int      `json:"sessionId"`
	Prompt    string   `json:"prompt"`
	ImgData   []string `json:"imgData"`
	ImgMimes  []string `json:"imgDataMimes"`
	ImgNames  []string `json:"imgDataNames"`
	SearchWeb bool     `json:"searchWeb"`
}
