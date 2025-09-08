package models

type ClientSignals struct {
	SessionId         int      `json:"sessionId"`
	Prompt            string   `json:"prompt"`
	FileData          []string `json:"fileData"`
	FileDataMimes     []string `json:"fileDataMimes"`
	FileDataNames     []string `json:"fileDataNames"`
	SearchWeb         bool     `json:"searchWeb"`
	FileId            string   `json:"fileId"`
	SessionIdToDelete int      `json:"sessionIdToDelete"`
}

type FileDataDisplay struct {
	FileName string
	FileData string
	IsImg    bool
}
