package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetOpenAIEmbeddings(text string, channel chan<- []float32) {
	url := os.Getenv("OPENAI_API_EMBEDDING_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_EMBEDDING_MODEL")

	requestBody := models.OpenAIEmbeddingRequest{
		Input: text,
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for embedding request: %v\n", err)
		channel <- nil
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for embedding request: %v\n", err)
		channel <- nil
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai embedding: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- nil
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	// fmt.Printf("OpenAI Embedding response: %v\n", string(responseData))
	var embeddingResponse models.OpenAIEmbeddingResponse
	err = json.Unmarshal(responseData, &embeddingResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for embedding request: %v\n", err)
		channel <- nil
		return
	}
	if len(embeddingResponse.Data) == 0 {
		fmt.Printf("No embedding data found in response, response body: %v\n", string(responseData))
		channel <- nil
		return
	}
	channel <- embeddingResponse.Data[0].Embedding
}
