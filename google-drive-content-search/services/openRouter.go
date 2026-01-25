package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"google-drive-content-search/models"
	"io"
	"net/http"
	"os"
)

func CallOpenRouter(aiRequest models.OpenRouterRequest, channel chan<- models.OpenRouterResponse) {
	url := os.Getenv("OPEN_ROUTER_API_URL")
	key := os.Getenv("OPEN_ROUTER_API_KEY")
	aiRequest.Model = os.Getenv("OPEN_ROUTER_MODEL")
	defaultVal := models.OpenRouterResponse{}
	aiRequestBytes, err := json.Marshal(aiRequest)
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error marshaling request to call Open Router: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error in making openrouter message api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Error in making openrouter message api call %v\n", string(respBody))
		}
		channel <- defaultVal
		return
	}
	respBody, err := io.ReadAll(response.Body)
	// fmt.Printf("Non-streaming response body OpenRouter API call: %s\n", string(respBody))
	if err != nil {
		fmt.Printf("Error reading non-streaming response body OpenRouter API call: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	var nonStreamResponse models.OpenRouterResponse
	err = json.Unmarshal(respBody, &nonStreamResponse)
	if err != nil {
		fmt.Printf("Error unmarshaling non-streaming response OpenRouter API call: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	channel <- nonStreamResponse

}
