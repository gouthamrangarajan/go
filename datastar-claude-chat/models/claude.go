package models

import (
	"encoding/json"
	"strings"
)

type ClaudeRequestImageContentSource struct {
	Type      string `json:"type,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}
type ClaudeRequestImageContent struct {
	Type   string                          `json:"type"`
	Source ClaudeRequestImageContentSource `json:"source,omitempty"`
	Text   string                          `json:"text,omitempty"`
}

type ClaudeRequestMessage struct {
	Role             string                      `json:"role"`
	Content          string                      `json:",omitempty"`
	ContentWithImage []ClaudeRequestImageContent `json:",omitempty"`
}
type ClaudeRequestTools struct {
	Type    string `json:"type,omitempty"`
	Name    string `json:"name,omitempty"`
	MaxUses int    `json:"max_uses,omitempty"`
}
type ClaudeRequest struct {
	Model       string                 `json:"model"`
	Messages    []ClaudeRequestMessage `json:"messages"`
	MaxToken    int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
	Tools       []ClaudeRequestTools   `json:"tools,omitempty"`
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

func (requestMessage ClaudeRequestMessage) MarshalJSON() ([]byte, error) {
	output := make(map[string]interface{})
	output["role"] = requestMessage.Role
	var outErr error
	var outputBytes []byte
	if len(requestMessage.ContentWithImage) > 0 {
		if bytes, err := json.Marshal(requestMessage.ContentWithImage); err == nil {
			output["content"] = string(bytes)
			// output["content"] = strings.ReplaceAll(output["content"].(string), ",\"source\":{}", "")
		} else {
			outErr = err
		}
	} else if strings.TrimSpace(requestMessage.Content) != "" {
		output["content"] = requestMessage.Content
	}
	if outErr == nil {
		outputBytes, outErr = json.Marshal(output)
	}
	return outputBytes, outErr
}
