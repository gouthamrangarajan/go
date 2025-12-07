package models

type ClientSignals struct {
	SessionId int    `json:"sessionId"`
	Prompt    string `json:"prompt"`
	FileData  []struct {
		Name     string `json:"name"`
		Contents string `json:"contents"`
		Mime     string `json:"mime"`
	} `json:"fileData"`
	SearchWeb bool `json:"searchWeb"`
	// FileId            string   `json:"fileId"`
	SessionIdToDelete int    `json:"sessionIdToDelete"`
	MenuSearchTerm    string `json:"menuSrchTxt"`
}

type FileDataDisplay struct {
	FileName string
	FileData string
	IsImg    bool
}

type UICookie struct {
	Name  string
	Value string
}

type PromptRequest struct {
	SessionId int
	Prompt    string
	// PromptFileId string
	SearchWeb     bool
	FileMediaType string
	FileData      string
}
