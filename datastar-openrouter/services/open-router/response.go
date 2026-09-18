package openrouter

type StreamResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Images  []struct {
				Type     string `json:"type,omitempty"`
				ImageUrl struct {
					Url string `json:"url,omitempty"`
				} `json:"image_url,omitempty"`
			} `json:"images,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
}
type Response struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Images  []struct {
				Type     string `json:"type,omitempty"`
				ImageUrl struct {
					Url string `json:"url,omitempty"`
				} `json:"image_url,omitempty"`
			} `json:"images,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}
