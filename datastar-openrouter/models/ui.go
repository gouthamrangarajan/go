package models

type ClientSignals struct {
	Prompt            string `json:"prompt"`
	SessionId         int    `json:"sessionId"`
	ModelId           string `json:"modelId"`
	SessionIdToDelete int    `json:"sessionIdToDelete"`
	FileData          []struct {
		Name     string `json:"name"`
		Contents string `json:"contents"`
		Mime     string `json:"mime"`
	} `json:"fileData"`
	WebSearch             bool   `json:"webSearch"`
	ImageGeneration       bool   `json:"imageGeneration"`
	MessageIdToRetry      int    `json:"messageIdToRetry"`
	SearchMenu            string `json:"searchMenu"`
	UiSid                 string `json:"uiSid"`
	MessageIdToFetchImage int    `json:"messageIdToFetchImage"`
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

type UIMainModel struct {
	Messages         []ChatConversation
	Sessions         []ChatSession
	AIModels         []AIModel
	AllowWebSearch   bool
	ImageGeneration  bool
	CurrentSessionId int
	MenuSearchTerm   string
	UiSid            string
}

type UIToolTipModel struct {
	DataShowAttribute string
	Text              string
	PostionAnchor     string
	AnchorStyle       string
}
