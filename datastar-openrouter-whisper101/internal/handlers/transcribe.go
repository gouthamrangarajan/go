package handlers

import (
	"datastar-openrouter-whisper101/internal/services/openrouter"
	"datastar-openrouter-whisper101/internal/views/components"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"
)

type OpenRouterClient interface {
	TranscribeAudio(request *openrouter.TranscribeRequest, resultChan chan *openrouter.TranscribeResult)
}

type TranscribeHandler struct {
	client OpenRouterClient
}

func NewTranscribeHandler(openRouterClient OpenRouterClient) *TranscribeHandler {
	return &TranscribeHandler{
		client: openRouterClient,
	}
}

func (t *TranscribeHandler) Transcribe(responseWriter http.ResponseWriter, request *http.Request) {
	request.ParseMultipartForm(10 << 20) // Limit upload size to 10MB
	file, _, err := request.FormFile("audioFile")
	if err != nil {
		http.Error(responseWriter, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.Transcribing(), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
	defer file.Close()
	// test, _ := io.ReadAll(file)
	// os.WriteFile("test.webm", test, 0644)
	// Here you can process the uploaded file (e.g., save it, transcribe it, etc.)
	// For demonstration, we'll just send a success response.
	// responseWriter.WriteHeader(http.StatusOK)
	// responseWriter.Write([]byte("File uploaded successfully"))
	transcribeRequest := openrouter.NewTranscribeRequest()
	transcribeRequest.Audio.Base64Data, transcribeRequest.Audio.Format, err = readFileAsBase64AndFormat(file)
	if err != nil {
		fmt.Printf("Error reading the file: %v\n", err)
		sse.PatchElementTempl(components.TranscribeError(), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
		return
	}
	resultChan := make(chan *openrouter.TranscribeResult, 1)
	go t.client.TranscribeAudio(transcribeRequest, resultChan)
	transcribeResult := <-resultChan
	if transcribeResult.Error != nil {
		fmt.Printf("Error transcribing the audio: %v\n", transcribeResult.Error)
		sse.PatchElementTempl(components.TranscribeError(), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
		return
	}
	sse.PatchElementTempl(components.TranscribeResult(&transcribeResult.Response), datastar.WithModeOuter(), datastar.WithUseViewTransitions(true))
}

func readFileAsBase64AndFormat(file multipart.File) (string, string, error) {
	// Read the file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", "", err
	}

	// Encode the file content to base64
	base64Data := base64.StdEncoding.EncodeToString(fileBytes)

	// Determine the file format (you can enhance this logic based on your needs)
	format := http.DetectContentType(fileBytes)

	return base64Data, format, nil
}
