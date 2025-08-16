package models

type ClaudeRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model    string                 `json:"model"`
	Messages []ClaudeRequestMessage `json:"messages"`
	MaxToken int                    `json:"max_tokens,omitempty"`
	Stream   bool                   `json:"stream,omitempty"`
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
