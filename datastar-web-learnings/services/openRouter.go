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

func VerifyTechnologyTopicsSearchAndOptimizeQueryUsingOpenRouter(query string, channel chan<- string) {
	url := os.Getenv("OPENROUTER_API_URL")
	key := os.Getenv("OPENROUTER_API_KEY")
	responseVal := models.OpenRouterResponse{}
	aiRequestBytes, err := json.Marshal(models.OpenRouterRequest{
		Model: os.Getenv("OPENROUTER_API_MODEL"),
		Messages: []models.OpenRouterRequestMessage{
			{
				Role:    "user",
				Content: fmt.Sprintf(PROMPT2_TO_CHECK_TECH_RELATED_SEARCH, query),
			},
		},
	})
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
		channel <- ""
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- ""
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- ""
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error in making openrouter message api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Error in making openrouter message api call %v\n", string(respBody))
		}
		channel <- ""
		return
	}

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body OpenRouter API call: %v\n", err.Error())
		channel <- ""
		return
	}

	err = json.Unmarshal(respBody, &responseVal)
	if err != nil {
		fmt.Printf("Error unmarshaling response OpenRouter API call: %v\n", err.Error())
		channel <- ""
		return
	}
	if len(responseVal.Choices) > 0 {
		content := responseVal.Choices[0].Message.Content
		contents := strings.SplitN(content, "\n", 2)
		if len(contents) > 1 {
			// fmt.Printf("Sending message to channel: %v\n", contents[1])
			channel <- strings.TrimSpace(contents[1])
			return
		}
	} else {
		fmt.Printf("No choices in response OpenRouter API call\n")
	}
	channel <- ""
}
