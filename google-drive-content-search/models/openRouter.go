package models

type OpenRouterResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
type OpenRouterRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterRequest struct {
	Model    string                     `json:"model"`
	Messages []OpenRouterRequestMessage `json:"messages"`
}
