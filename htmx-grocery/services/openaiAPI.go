package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAIAPIRequestMessageField struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIAPIRequest struct {
	Model    string                         `json:"model"`
	Messages []OpenAIAPIRequestMessageField `json:"messages"`
}

type response struct {
	Id      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func CallOpenAI(url string, key string, request OpenAIAPIRequest) string {
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to JSON in Open AI call: %v+\n", err)
		return ""
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request in Open AI call: %v+\n", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{
		// Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error getting response in Open AI call: %v+\n", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Open AI call: %v+\n", resp.Status)
		} else {
			fmt.Printf("Error in Open AI call: %v+\n", string(errorMsg))
		}
		return ""
	}
	var parsedResponse response
	err = json.NewDecoder(resp.Body).Decode(&parsedResponse)
	if err != nil {
		fmt.Printf("Error parsing response in Open AI call: %v+\n", err)
	}
	return parsedResponse.Choices[0].Message.Content
}

func CallOpenAIViaChannel(url string, key string, request OpenAIAPIRequest, channel chan<- string) {
	channel <- CallOpenAI(url, key, request)
}

func CallOpenAIViaChannelTillContext(url string, key string, request OpenAIAPIRequest, channel chan<- string, ctx context.Context) {
	newChannel := make(chan string)
	defer close(newChannel)
	go func() {
		newChannel <- CallOpenAI(url, key, request)
	}()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Open AI API call ignore due to context done")
			channel <- ""
			return
		case valueInNewChannel := <-newChannel:
			channel <- valueInNewChannel
			return
		}
	}
}
