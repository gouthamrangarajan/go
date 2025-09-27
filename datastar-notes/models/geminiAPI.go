package models

type GeminiRequestParts struct {
	Text string `json:"text"`
}

type GeminiRequestContent struct {
	Role  string               `json:"role"`
	Parts []GeminiRequestParts `json:"parts"`
}

type GeminiRequestConfigThinking struct {
	Budget int8 `json:"thinkingBudget"`
}

type GeminiRequestConfig struct {
	Thinking GeminiRequestConfigThinking `json:"thinkingConfig,omitempty"`
}
type GeminiRequest struct {
	Contents []GeminiRequestContent `json:"contents"`
	Config   GeminiRequestConfig    `json:"generationConfig,omitempty"`
}

type GeminiResponseParts struct {
	Text *string `json:"text"`
}
type GeminiResponseContent struct {
	Role  string                `json:"role"`
	Parts []GeminiResponseParts `json:"parts"`
}
type GeminiResponse struct {
	Candidates []struct {
		Content GeminiResponseContent `json:"content"`
	} `json:"candidates"`
}
