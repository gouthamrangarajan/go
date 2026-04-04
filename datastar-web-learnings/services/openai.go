package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenAI(query string, channel chan<- string) {
	url := os.Getenv("OPENAI_API_URL")
	key := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_API_MODEL")

	requestBody := models.OpenAIRequest{
		Input: fmt.Sprintf(PROMPT2_TO_CHECK_TECH_RELATED_SEARCH, query),
		Model: model,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error marshalling request body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating HTTP request for open ai request: %v\n", err)
		channel <- ""
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for open ai request: %v\n", err)
		channel <- ""
		return
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling open ai request: %v\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(responseData))
		}
		channel <- ""
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	// fmt.Printf("OpenAI response: %v\n", string(responseData))
	var openaiResponse models.OpenAIResponse
	err = json.Unmarshal(responseData, &openaiResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for open ai request: %v\n", err)
		channel <- ""
		return
	}
	if len(openaiResponse.Output) == 0 {
		fmt.Printf("No output data found in open ai response, response body: %v\n", string(responseData))
		channel <- ""
		return
	}
	for _, output := range openaiResponse.Output {
		if output.Role == "assistant" {
			for _, content := range output.Content {
				contents := strings.SplitN(content.Text, "\n", 2)
				if len(contents) > 1 {
					// fmt.Printf("Sending message to channel: %v\n", contents[1])
					channel <- strings.TrimSpace(contents[1])
					return
				}
			}
		}
	}
	channel <- ""
}
