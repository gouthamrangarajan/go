package models

import (
	"encoding/json"
	"strings"
)

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

type OpenRouterRequest struct {
	Model    string                     `json:"model"`
	Messages []OpenRouterRequestMessage `json:"messages"`
	Stream   bool                       `json:"stream"`
}
type OpenRouterRequestMessageContentWithFileData struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageUrl struct {
		Url string `json:"url,omitempty"`
	} `json:"image_url,omitempty"`
	File struct {
		Name string `json:"filename,omitempty"`
		Data string `json:"file_data,omitempty"`
	} `json:"file,omitempty"`
}
type OpenRouterRequestMessage struct {
	Role                string `json:"role"`
	Content             string
	ContentWithFileData []OpenRouterRequestMessageContentWithFileData
}

func (requestMessage OpenRouterRequestMessage) MarshalJSON() ([]byte, error) {
	output := make(map[string]interface{})
	output["role"] = requestMessage.Role
	var outErr error
	var outputBytes []byte
	if len(requestMessage.ContentWithFileData) > 0 {
		output["content"] = []interface{}{}
		for _, contentPart := range requestMessage.ContentWithFileData {
			contentPartMap := make(map[string]interface{})
			contentPartMap["type"] = contentPart.Type
			if strings.TrimSpace(contentPart.ImageUrl.Url) != "" {
				contentPartMap["image_url"] = map[string]interface{}{
					"url": contentPart.ImageUrl.Url,
				}
			} else if strings.TrimSpace(contentPart.File.Data) != "" {
				contentPartMap["file"] = map[string]interface{}{
					"filename":  contentPart.File.Name,
					"file_data": contentPart.File.Data,
				}
			} else {
				contentPartMap["text"] = contentPart.Text
			}
			output["content"] = append(output["content"].([]interface{}), contentPartMap)
		}

	} else {
		output["content"] = requestMessage.Content
	}
	// fmt.Printf("Marshaling OpenRouterRequestMessage: %v\n", output)
	outputBytes, outErr = json.Marshal(output)
	return outputBytes, outErr
}
