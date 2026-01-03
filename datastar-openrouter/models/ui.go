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
