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

type GeminiRequest struct {
	Contents []GeminiRequestContent `json:"contents"`
	Config   struct {
		Thinking struct {
			Budget int8 `json:"thinkingBudget"`
		} `json:"thinkingConfig"`
	} `json:"generationConfig"`
}
type GeminiResponse struct {
	Candidates []struct {
		Content GeminiRequestContent
	} `json:"candidates"`
}
