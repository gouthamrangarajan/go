package openrouter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

type Client struct {
	apiKey             string
	audioTranscribeURL string
}
type TranscribeResult struct {
	Response TranscribeResponse
	Error    error
}

func NewClient() *Client {
	return &Client{
		apiKey:             os.Getenv("OPEN_ROUTER_API_KEY"),
		audioTranscribeURL: os.Getenv("OPEN_ROUTER_AUDIO_TRANSCRIBE_URL"),
	}
}

func (c *Client) TranscribeAudio(request *TranscribeRequest, resultChan chan *TranscribeResult) {
	defer close(resultChan)
	httpClient := &http.Client{}
	jsonRequestData, err := json.Marshal(request)
	if err != nil {
		resultChan <- &TranscribeResult{Error: err}
		return
	}
	httpRequest, err := http.NewRequest("POST", c.audioTranscribeURL, bytes.NewBuffer(jsonRequestData))
	if err != nil {
		resultChan <- &TranscribeResult{Error: err}
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		resultChan <- &TranscribeResult{Error: err}
		return
	}
	defer httpResponse.Body.Close()

	var transcribeResponse TranscribeResponse
	err = json.NewDecoder(httpResponse.Body).Decode(&transcribeResponse)
	if err != nil {
		resultChan <- &TranscribeResult{Error: err}
		return
	}
	resultChan <- &TranscribeResult{Response: transcribeResponse}
}
