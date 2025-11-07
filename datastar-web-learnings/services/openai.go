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

var prompt = `You will be given a search query from a user. Your task is to decide whether this query is related to technology or tech topics.

Consider these as related to technology:
- General tech concepts (e.g., programming, AI, blockchain, gadgets)
- Technology companies or products
- Names of technology personalities, developers, engineers, or influencers (e.g., "Evan You", "Dan Abramov")
- Technology events or conferences
- Anything involving software, hardware, coding, IT, or digital innovations

If the query is related to technology, respond with: YES  
If it is not related to technology, respond with: NO  

Do not provide any explanation or additional text, only YES or NO.

Example:  
Query: "Evan You"  
Answer: YES

Query: "Best cooking recipes"  
Answer: NO

Now, decide for this query:  
"%v"`

func GetOpenAIEmbeddings(text string, channel chan<- []float32) {
	url := os.Getenv("OPENAI_API_EMBEDDING_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_EMBEDDING_MODEL")

	requestBody := models.OpenAIRequest{
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

func VerifyTechnologyTopicsSearch(text string, channel chan<- bool) {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")

	requestBody := models.OpenAIRequest{
		Input: fmt.Sprintf(prompt, text),
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for open ai request: %v\n", err)
		channel <- false
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for open ai request: %v\n", err)
		channel <- false
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for open ai request: %v\n", err)
		channel <- false
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai request: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- false
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for open ai request: %v\n", err)
		channel <- false
		return
	}
	// fmt.Printf("OpenAI response: %v\n", string(responseData))
	var embeddingResponse models.OpenAIResponse
	err = json.Unmarshal(responseData, &embeddingResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for open ai request: %v\n", err)
		channel <- false
		return
	}
	if len(embeddingResponse.Output) == 0 {
		fmt.Printf("No output data found in open ai response, response body: %v\n", string(responseData))
		channel <- false
		return
	}
	for _, output := range embeddingResponse.Output {
		if output.Role == "assistant" {
			for _, content := range output.Content {
				if content.Text == "YES" {
					channel <- true
					return
				}
			}
		}
	}
	channel <- false
}
