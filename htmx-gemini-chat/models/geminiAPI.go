package models

type GeminiRequestParts struct {
	Text    *string                 `json:"text,omitempty"`
	ImgData *GeminiRequestImageData `json:"inline_data,omitempty"`
}
type GeminiRequestImageData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}
type GeminiRequestContent struct {
	Role  string               `json:"role"`
	Parts []GeminiRequestParts `json:"parts"`
}
type GeminiThinkingConfig struct {
	Budget int8 `json:"thinkingBudget"`
}
type GeminiGenerationConfig struct {
	Thinking GeminiThinkingConfig `json:"thinkingConfig"`
}
type GeminiRequest struct {
	Contents []GeminiRequestContent `json:"contents"`
	Config   GeminiGenerationConfig `json:"generationConfig"`
}
type GeminiResponse struct {
	Candidates []struct {
		Content GeminiRequestContent
	}
}
