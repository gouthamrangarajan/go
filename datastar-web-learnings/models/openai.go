package models

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}
