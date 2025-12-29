package models

type OpenRouterRequest struct {
	Model    string                     `json:"model"`
	Messages []OpenRouterRequestMessage `json:"messages"`
	Stream   bool                       `json:"stream"`
}
type OpenRouterRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type OpenRouterStreamResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
type OpenRouterModelIdAndDeltaString struct {
	DeltaContent string
	ModelId      string
}
