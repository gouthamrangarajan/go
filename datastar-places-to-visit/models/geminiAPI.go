package models

type GeminiRequestContentPart struct {
	Text *string `json:"text"`
}

type GeminiRequestContent struct {
	Role  string                     `json:"role"`
	Parts []GeminiRequestContentPart `json:"parts"`
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
