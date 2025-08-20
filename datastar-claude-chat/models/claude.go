package models

type ClaudeRequestImageContent struct {
	Type   string `json:"type"`
	Source struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
	Text string `json:"text,omitempty"`
}
type ClaudeRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
	// ContentArrayForImage []ClaudeRequestImageContent `json:"contentArrayForImage,omitempty"`
}

type ClaudeRequest struct {
	Model       string                 `json:"model"`
	Messages    []ClaudeRequestMessage `json:"messages"`
	MaxToken    int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
}

type ClaudeResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type ClaudeResponse struct {
	Role    string                  `json:"role"`
	Content []ClaudeResponseContent `json:"content"`
}

type ClaudeStreamingResponse struct {
	Type         string                `json:"type"`
	Delta        ClaudeResponseContent `json:"delta,omitempty"`
	ContentBlock ClaudeResponseContent `json:"content_block,omitempty"`
}
