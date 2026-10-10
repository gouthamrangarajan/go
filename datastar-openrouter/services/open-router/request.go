package openrouter

import (
	"encoding/json"
	"strings"
)

type CompletionsRequest struct {
	Model      string              `json:"model"`
	Messages   []RequestMessage    `json:"messages"`
	Stream     bool                `json:"stream"`
	Plugins    []map[string]string `json:"plugins,omitempty"`
	Modalities []string            `json:"modalities,omitempty"`
}
type RequestMessageContentWithFileData struct {
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
type RequestMessage struct {
	Role                string `json:"role"`
	Content             string
	ContentWithFileData []RequestMessageContentWithFileData
}

func (requestMessage RequestMessage) MarshalJSON() ([]byte, error) {
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
	// fmt.Printf("Marshaling RequestMessage: %v\n", output)
	outputBytes, outErr = json.Marshal(output)
	return outputBytes, outErr
}

type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Stream         bool   `json:"stream"`
	InputReference []struct {
		Type     string `json:"type"`
		ImageUrl struct {
			Url string `json:"url"`
		} `json:"image_url"`
	} `json:"input_reference,omitempty"`
}
