package openrouter

import "os"

type TranscribeRequest struct {
	Model string `json:"model"`
	Audio struct {
		Base64Data string `json:"data"`
		Format     string `json:"format"`
	} `json:"input_audio"`
}

func NewTranscribeRequest() *TranscribeRequest {
	return &TranscribeRequest{
		Model: os.Getenv("OPEN_ROUTER_AUDIO_TRANSCRIBE_MODEL"),
	}
}
