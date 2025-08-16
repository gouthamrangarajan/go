package services

import (
	"bufio"
	"bytes"
	"datastar-claude-chat/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func CallClaudeAPI(prompt string, channel chan string) {
	defer close(channel)
	maxTokenStr := os.Getenv("CLAUDE_MAX_TOKEN")
	maxToken, err := strconv.Atoi(maxTokenStr)
	if err != nil {
		maxToken = 8000
	}
	aiRequest := models.ClaudeRequest{
		Model:    os.Getenv("CLAUDE_MODEL"),
		Messages: []models.ClaudeRequestMessage{},
		MaxToken: maxToken,
	}
	aiRequest.Messages = append(aiRequest.Messages, models.ClaudeRequestMessage{
		Role:    "user",
		Content: prompt,
	})
	aiRequestBytes, err := json.Marshal(aiRequest)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", os.Getenv("CLAUDE_API_URL"), bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	httpRequest.Header.Set("x-api-key", os.Getenv("CLAUDE_API_KEY"))
	httpRequest.Header.Set("anthropic-version", os.Getenv("CLAUDE_API_HEADER_VERSION"))
	httpRequest.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error in making claude api call: received status code %d\n", resp.StatusCode)
		if err != nil {
			fmt.Printf("Error in making claude api call %v\n", string(respBody))
		}
		channel <- "Error"
		return
	}
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err.Error())
		channel <- "Error"
		return
	}
	var claudeResponse models.ClaudeResponse
	err = json.Unmarshal(respBody, &claudeResponse)
	if err != nil {
		fmt.Printf("Error decoding response: %v\n", err.Error())
		channel <- "Error"
		return
	}

	for _, content := range claudeResponse.Content {
		if content.Type == "text" {
			channel <- content.Text
		} else {
			fmt.Printf("Unsupported content type: %s\n", content.Type)
		}
	}
}
func CallClaudeAPIStreaming(prompt string, channel chan string) {
	defer close(channel)
	maxTokenStr := os.Getenv("CLAUDE_MAX_TOKEN")
	maxToken, err := strconv.Atoi(maxTokenStr)
	if err != nil {
		maxToken = 8000
	}
	aiRequest := models.ClaudeRequest{
		Model:    os.Getenv("CLAUDE_MODEL"),
		Messages: []models.ClaudeRequestMessage{},
		MaxToken: maxToken,
		Stream:   true,
	}
	aiRequest.Messages = append(aiRequest.Messages, models.ClaudeRequestMessage{
		Role:    "user",
		Content: prompt,
	})
	aiRequestBytes, err := json.Marshal(aiRequest)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", os.Getenv("CLAUDE_API_URL"), bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	httpRequest.Header.Set("x-api-key", os.Getenv("CLAUDE_API_KEY"))
	httpRequest.Header.Set("anthropic-version", os.Getenv("CLAUDE_API_HEADER_VERSION"))
	httpRequest.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error in making claude api call: received status code %d\n", resp.StatusCode)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in making claude api call %v\n", string(respBody))
		}
		channel <- "Error"
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "data: ") {
			line = strings.TrimSpace(strings.ReplaceAll(line, "data: ", ""))

			var claudeStreamingResponse models.ClaudeStreamingResponse
			err := json.Unmarshal([]byte(line), &claudeStreamingResponse)
			if err != nil {
				fmt.Printf("Error decoding response: %v, %v\n", line, err.Error())
			} else {
				switch {
				case claudeStreamingResponse.Type == "content_block_delta" &&
					claudeStreamingResponse.Delta.Type == "text_delta":
					channel <- claudeStreamingResponse.Delta.Text
				}
			}

		}
	}
}
func CallClaudeAPIStreamingWithRequest(aiRequest models.ClaudeRequest, channel chan string) {
	defer close(channel)

	aiRequestBytes, err := json.Marshal(aiRequest)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", os.Getenv("CLAUDE_API_URL"), bytes.NewBuffer(aiRequestBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	httpRequest.Header.Set("x-api-key", os.Getenv("CLAUDE_API_KEY"))
	httpRequest.Header.Set("anthropic-version", os.Getenv("CLAUDE_API_HEADER_VERSION"))
	httpRequest.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error in making claude api call: received status code %d\n", resp.StatusCode)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in making claude api call %v\n", string(respBody))
		}
		channel <- "Error"
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "data: ") {
			line = strings.TrimSpace(strings.ReplaceAll(line, "data: ", ""))

			var claudeStreamingResponse models.ClaudeStreamingResponse
			err := json.Unmarshal([]byte(line), &claudeStreamingResponse)
			if err != nil {
				fmt.Printf("Error decoding response: %v, %v\n", line, err.Error())
			} else {
				switch {
				case claudeStreamingResponse.Type == "content_block_delta" &&
					claudeStreamingResponse.Delta.Type == "text_delta":
					channel <- claudeStreamingResponse.Delta.Text
				}
			}

		}
	}
}
