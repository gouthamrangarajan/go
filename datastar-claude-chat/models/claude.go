package models

import (
	"encoding/json"
	"strings"
)

type ClaudeRequestFileContentSource struct {
	Type   string `json:"type,omitempty"`
	FileId string `json:"file_id,omitempty"`
}
type ClaudeRequestFileContent struct {
	Type   string                         `json:"type"`
	Source ClaudeRequestFileContentSource `json:"source,omitempty"`
	Text   string                         `json:"text,omitempty"`
}
type ClaudeFileUploadResponse struct {
	Id string `json:"id"`
}

type ClaudeRequestImageInlineContentSource struct {
	Type      string `json:"type,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}
type ClaudeRequestImageInlineContent struct {
	Type   string                                 `json:"type"`
	Source *ClaudeRequestImageInlineContentSource `json:"source,omitempty"`
	Text   string                                 `json:"text,omitempty"`
}
type ClaudeRequestMessage struct {
	Role             string `json:"role"`
	Content          string
	ContentWithImage []ClaudeRequestImageInlineContent
	// ContentWithFile []ClaudeRequestFileContent `json:",omitempty"`
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
