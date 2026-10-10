package openrouter

import (
	"bufio"
	"bytes"
	"datastar-openrouter/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Client struct {
	url                string
	key                string
	embeddingUrl       string
	imageGenerationUrl string
}

func NewClient() *Client {
	return &Client{
		url:                os.Getenv("OPEN_ROUTER_API_URL"),
		key:                os.Getenv("OPEN_ROUTER_API_KEY"),
		embeddingUrl:       os.Getenv("OPEN_ROUTER_EMBEDDING_URL"),
		imageGenerationUrl: os.Getenv("OPEN_ROUTER_IMAGE_GENERATION_URL"),
	}
}

func (c *Client) CreateChatCompletion(aiRequest CompletionsRequest, channel chan<- models.OpenRouterModelIdAndDeltaString) {
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
	httpRequest, err := http.NewRequest("POST", c.url, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
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
		var nonStreamResponse CompletionsResponse
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
		// fmt.Printf("Received line from stream: %s\n", scanner.Text())
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
			var streamResponse CompletionsStreamResponse
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
	// Check for errors after the loop
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading streaming response: %v\n", err.Error())
		channel <- defaultVal
	}
}

func (c *Client) CreateEmbedding(embeddingRequest EmbeddingRequest, channel chan<- EmbeddingResponse) {
	defer close(channel)
	returnVal := EmbeddingResponse{}

	requestBytes, err := json.Marshal(embeddingRequest)
	if err != nil {
		fmt.Printf("Error marshaling embedding request: %v\n", err.Error())
		channel <- returnVal
		return
	}
	httpRequest, err := http.NewRequest("POST", c.embeddingUrl, bytes.NewBuffer(requestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request for embedding: %v\n", err.Error())
		channel <- returnVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
	httpRequest.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for embedding: %v\n", err.Error())
		channel <- returnVal
		return
	}
	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error in making openrouter embedding api call: received status code %d\n", response.StatusCode)
		if err == nil {
			fmt.Printf("Error in making openrouter embedding api call %v\n", string(respBody))
		}
		channel <- returnVal
		return
	}
	// fmt.Printf("Embedding response body: %s\n", string(respBody)[0:500])
	err = json.Unmarshal(respBody, &returnVal)
	if err != nil {
		fmt.Printf("Error unmarshaling embedding response: %v\n", err.Error())
		channel <- returnVal
		return
	}
	channel <- returnVal

}

func (c *Client) CreateImageGeneration(aiRequest ImageGenerationRequest, channel chan<- models.OpenRouterModelIdAndDeltaString) {
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
	httpRequest, err := http.NewRequest("POST", c.imageGenerationUrl, bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.key))
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- defaultVal
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error in making openrouter image generation api call: received status code %d\n", response.StatusCode)
		respBody, err := io.ReadAll(response.Body)
		if err == nil {
			fmt.Printf("Error in making openrouter image generation api call %v\n", string(respBody))
		}
		channel <- defaultVal
		return
	}
	// fmt.Printf("number of previous image data %v\n", len(aiRequest.InputReference))
	if aiRequest.Stream == false {
		respBody, err := io.ReadAll(response.Body)
		// fmt.Printf("Non-streaming response body OpenRouter image generation API call: %s\n", string(respBody[0:500]))
		if err != nil {
			fmt.Printf("Error reading non-streaming response body OpenRouter image generation API call: %v\n", err.Error())
			channel <- defaultVal
			return
		}
		var nonStreamResponse ImageGenerationResponse
		err = json.Unmarshal(respBody, &nonStreamResponse)
		if err != nil {
			fmt.Printf("Error unmarshaling non-streaming response OpenRouter image generation API call: %v\n", err.Error())
			channel <- defaultVal
			return
		}
		if len(nonStreamResponse.Data) > 0 {
			if strings.TrimSpace(nonStreamResponse.Data[0].MediaType) == "" {
				nonStreamResponse.Data[0].MediaType = "image/png"
			}
			channel <- models.OpenRouterModelIdAndDeltaString{DeltaContent: "", ModelId: aiRequest.Model,
				DeltaImage: "data:" + nonStreamResponse.Data[0].MediaType + ";base64," + nonStreamResponse.Data[0].B64Json}
			return
		} else {
			fmt.Printf("No data in non-streaming response OpenRouter image generation API call\n")
			channel <- defaultVal
			return
		}
	}
	scanner := bufio.NewScanner(response.Body)
	line := ""

	for scanner.Scan() {
		// fmt.Printf("Received line from stream: %s\n", scanner.Text())
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
			var streamResponse ImageGenerationStreamingResponse
			err = json.Unmarshal([]byte(line), &streamResponse)
			if err != nil {
				fmt.Printf("Error unmarshaling stream response: %v\n", err.Error())
				// channel <- "Error"
				// return
			} else {
				if streamResponse.B64Json != "" {
					// fmt.Println("Sending content to channel:", content)
					if strings.TrimSpace(streamResponse.MediaType) == "" {
						streamResponse.MediaType = "image/png"
					}
					channel <- models.OpenRouterModelIdAndDeltaString{DeltaContent: "", ModelId: aiRequest.Model,
						DeltaImage: "data:" + streamResponse.MediaType + ";base64," + streamResponse.B64Json}
				}

				line = ""
			}
		}
	}
	// Check for errors after the loop
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading streaming response: %v\n", err.Error())
		channel <- defaultVal
	}
}
