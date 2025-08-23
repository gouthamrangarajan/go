package services

import (
	"bufio"
	"bytes"
	"datastar-claude-chat/models"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func CallClaudeAPI(aiRequest models.ClaudeRequest, channel chan string) {
	defer close(channel)

	aiRequestBytes, err := json.Marshal(aiRequest)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	// fmt.Printf("request to ai: %v\n", string(aiRequestBytes))
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
		// fmt.Println(line)
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
				case claudeStreamingResponse.Type == "error":
					channel <- "Error"
				}
			}

		}
	}
}

func CallClaudeAPIFileUpload(base64Data string, channel chan string) {
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		fmt.Printf("Error decoding base64 data")
		channel <- "Error"
		return
	}
	client := &http.Client{}
	httpRequest, err := http.NewRequest("POST", os.Getenv("CLAUDE_FILE_UPLOAD_API_URL"), bytes.NewBuffer(decodedBytes))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	httpRequest.Header.Set("x-api-key", os.Getenv("CLAUDE_API_KEY"))
	httpRequest.Header.Set("anthropic-version", os.Getenv("CLAUDE_API_HEADER_VERSION"))
	httpRequest.Header.Set("anthropic-beta", "CALUDE_API_HEADER_FILE_UPLOAD")
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err.Error())
		channel <- "Error"
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	respBodyStr := string(respBody)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error in making claude file upload api call: received status code %d\n", resp.StatusCode)
		if err != nil {
			fmt.Printf("Error in making claude file upload api call %v\n", respBodyStr)
		}
		channel <- "Error"
		return
	}

	fmt.Printf("File upload API response %v\n", respBodyStr)
	channel <- respBodyStr
}

//TO debug
//  clonedMessageImageContent := []models.ClaudeRequestImageContent{}
// 	if len(aiRequest.Messages) > 0 {
// 		if len(aiRequest.Messages[0].ContentWithImage) > 0 {
// 			clonedMessageImageContent = append(clonedMessageImageContent, models.ClaudeRequestImageContent{
// 				Type: aiRequest.Messages[0].ContentWithImage[0].Type,
// 				Source: models.ClaudeRequestImageContentSource{
// 					Data:      aiRequest.Messages[0].ContentWithImage[0].Source.Data[0:50],
// 					Type:      aiRequest.Messages[0].ContentWithImage[0].Source.Type,
// 					MediaType: aiRequest.Messages[0].ContentWithImage[0].Source.MediaType,
// 				},
// 			})
// 			clonedMessageImageContent = append(clonedMessageImageContent, models.ClaudeRequestImageContent{
// 				Type: aiRequest.Messages[0].ContentWithImage[1].Type,
// 				Text: aiRequest.Messages[0].ContentWithImage[1].Text,
// 			})
// 		}
// 	}

// 	aiRequestCloned := models.ClaudeRequest{}
// 	aiRequestCloned.Model = aiRequest.Model
// 	aiRequestCloned.MaxToken = aiRequest.MaxToken
// 	aiRequestCloned.Stream = aiRequest.Stream
// 	aiRequestCloned.Temperature = aiRequest.Temperature
// 	aiRequestCloned.Tools = aiRequest.Tools
// 	aiRequestCloned.Messages = []models.ClaudeRequestMessage{}
// 	aiRequestCloned.Messages = append(aiRequestCloned.Messages, models.ClaudeRequestMessage{
// 		ContentWithImage: clonedMessageImageContent,
// 		Role:             "user",
// 	})
// 	aiRequestCloneBytes, _ := json.Marshal(aiRequestCloned)
// 	fmt.Printf("Request to claude trimmed %v\n:", string(aiRequestCloneBytes))
