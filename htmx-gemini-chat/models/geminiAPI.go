package models

type GeminiRequestParts struct {
	Text string `json:"text"`
}
type GeminiRequestContent struct {
	Role  string               `json:"role"`
	Parts []GeminiRequestParts `json:"parts"`
}
type GeminiRequest struct {
	Contents []GeminiRequestContent `json:"contents"`
}
type GeminiResponse struct {
	Candidates []struct {
		Content GeminiRequestContent
	}
}
