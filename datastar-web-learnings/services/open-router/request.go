package openrouter

type Request struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
}

type RequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
