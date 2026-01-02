package models

type ClientSignals struct {
	Prompt            string `json:"prompt"`
	SessionId         int    `json:"sessionId"`
	ModelId           string `json:"modelId"`
	SessionIdToDelete int    `json:"sessionIdToDelete"`
}

type UICookie struct {
	Name  string
	Value string
}
