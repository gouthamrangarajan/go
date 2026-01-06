package services

import (
	"bufio"
	"bytes"
	"datastar-openrouter/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func CallOpenRouter(aiRequest models.OpenRouterRequest, channel chan<- models.OpenRouterModelIdAndDeltaString) {
	url := os.Getenv("OPEN_ROUTER_API_URL")
	key := os.Getenv("OPEN_ROUTER_API_KEY")
	defer close(channel)
	defaultVal := models.OpenRouterModelIdAndDeltaString{DeltaContent: "Error"}
	aiRequestBytes, err := json.Marshal(aiRequest)
	// fmt.Printf("OpenRouter Request:%v\n", string(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
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
	if aiRequest.Stream == false {
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
		if len(nonStreamResponse.Choices) > 0 {
			content := nonStreamResponse.Choices[0].Message.Content
			imgData := ""
			if len(nonStreamResponse.Choices[0].Message.Images) > 0 &&
				nonStreamResponse.Choices[0].Message.Images[0].ImageUrl.Url != "" {
				imgData = nonStreamResponse.Choices[0].Message.Images[0].ImageUrl.Url
			}
			channel <- models.OpenRouterModelIdAndDeltaString{DeltaContent: content, ModelId: nonStreamResponse.Model, DeltaImage: imgData}
			return
		} else {
			fmt.Printf("No choices in non-streaming response OpenRouter API call\n")
			channel <- defaultVal
			return
		}
	}
	scanner := bufio.NewScanner(response.Body)
	line := ""

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == ": OPENROUTER PROCESSING" {
			continue
		}

		line += scanner.Text()
		// if len(line) > 500 {
		// 	fmt.Printf("read line substring %v\n", line[500:])
		// } else {
		// 	fmt.Printf("read line %v\n", line)
		// }
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
			if line == "[DONE]" {
				break
			}
			var streamResponse models.OpenRouterStreamResponse
			err = json.Unmarshal([]byte(line), &streamResponse)
			if err != nil {
				fmt.Printf("Error unmarshaling stream response: %v\n", err.Error())
				// channel <- "Error"
				// return
			} else {
				if len(streamResponse.Choices) > 0 {
					content := streamResponse.Choices[0].Delta.Content
					imgData := ""
					if len(streamResponse.Choices[0].Delta.Images) > 0 &&
						streamResponse.Choices[0].Delta.Images[0].ImageUrl.Url != "" {
						imgData = streamResponse.Choices[0].Delta.Images[0].ImageUrl.Url
					}
					if content != "" || imgData != "" {
						// fmt.Println("Sending content to channel:", content)
						channel <- models.OpenRouterModelIdAndDeltaString{DeltaContent: content, ModelId: streamResponse.Model, DeltaImage: imgData}
					}
				}
				line = ""
			}
		}
	}
}
