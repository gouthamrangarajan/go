package services

import (
	"bytes"
	"datastar-portfolio/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func GetOpenRouterSuggestions(userQuery string, tagsFromDb []string, channel chan<- models.SearchSuggestion) {
	retVal := models.SearchSuggestion{}

	prompt := fmt.Sprintf(`You are a professional portfolio navigator for Goutham Rangarajan.
				Your Task: The user searched for '%s', but no projects matched. You must suggest the most relevant category from this list: '%s'.

				Rules:

				DO NOT give general technical advice or definitions.
				DO NOT suggest technologies NOT in the list.
				Write a 'Reason' that explains why the suggested tag is a good alternative to the user's query.
				Example: Query: "How to handle animations?" Result: "I don't have a specific animation project, but I use Framer Motion and Tailwind in my frontend work to create smooth interfaces.|Framer Motion"`,
		userQuery, strings.Join(tagsFromDb, ", "))

	url := os.Getenv("OPENROUTER_API_URL")
	key := os.Getenv("OPENROUTER_API_KEY")
	responseVal := models.OpenRouterResponse{}
	aiRequestBytes, err := json.Marshal(models.OpenRouterRequest{
		Model: os.Getenv("OPENROUTER_API_MODEL"),
		Messages: []models.OpenRouterRequestMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	})
	if err != nil {
		fmt.Printf("Error marshaling request to openRouter: %v\n", err.Error())
		channel <- retVal
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request to openRouter: %v\n", err.Error())
		channel <- retVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request to openRouter: %v\n", err.Error())
		channel <- retVal
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error calling openrouter message api: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Error calling making openrouter message api  %v\n", string(respBody))
		}
		channel <- retVal
		return
	}

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}

	err = json.Unmarshal(respBody, &responseVal)
	if err != nil {
		fmt.Printf("Error unmarshaling response OpenRouter API call: %v\n", err.Error())
		channel <- retVal
		return
	}
	if len(responseVal.Choices) > 0 {
		// fmt.Printf("OpenRouter API call returned content: %s\n", responseVal.Choices[0].Message.Content)
		// Split the content into suggestion and tag
		content := responseVal.Choices[0].Message.Content
		contentSplit := strings.Split(content, "|")
		if len(contentSplit) == 2 {
			retVal.Suggestion = strings.TrimSpace(contentSplit[0])
			retVal.Tag = strings.TrimSpace(contentSplit[1])
		}
		channel <- retVal
		return
	} else {
		fmt.Printf("No choices returned from OpenRouter API call\n")
		channel <- retVal
		return
	}

}
