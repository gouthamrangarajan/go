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
	WebSearch        bool `json:"webSearch"`
	ImageGeneration  bool `json:"imageGeneration"`
	MessageIdToRetry int  `json:"messageIdToRetry"`
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
}
