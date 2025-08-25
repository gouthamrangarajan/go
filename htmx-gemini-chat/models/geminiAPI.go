package models

type GeminiRequestParts struct {
	Text     *string                `json:"text,omitempty"`
	FileData *GeminiRequestFileData `json:"inline_data,omitempty"`
}
type GeminiRequestFileData struct {
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
	Tools map[string]interface{} `json:"tools,omitempty"`
}
type GeminiResponse struct {
	Candidates []struct {
		Content GeminiRequestContent
	} `json:"candidates"`
}
type GeminiEmbeddingRequestConfig struct {
	OutputDimension int `json:"output_dimensionality"`
}
type GeminiEmbeddingRequest struct {
	Model   string               `json:"model"`
	Content GeminiRequestContent `json:"content"`
	// Config  GeminiEmbeddingRequestConfig `json:"embedding_config"`
}

type GeminiEmbeddingResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}
