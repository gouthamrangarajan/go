package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func getSessionId(userId string, title string) int64 {
	var sessionId int64 = 0
	sessionChannel := make(chan []models.ChatSession)
	defer close(sessionChannel)
	go GetChatSessions(userId, sessionChannel)
	sessions := <-sessionChannel
	if len(sessions) == 0 {
		insertSessionChannel := make(chan int64)
		defer close(insertSessionChannel)
		go InsertChatSession(userId, title, insertSessionChannel)
		sessionId = <-insertSessionChannel
	} else {
		sessionId = sessions[0].Id //RG revisit for multiple session
	}
	return sessionId
}
func generateGeminiRequest(userId string, sessionId int64, prompt string) models.GeminiRequest {
	conversationsChannel := make(chan []models.ChatConversation)
	defer close(conversationsChannel)
	go GetChatConversations(userId, sessionId, conversationsChannel)
	conversations := <-conversationsChannel
	geminiRequest := models.GeminiRequest{}
	geminiRequest.Contents = make([]models.GeminiRequestContent, 0, len(conversations)+1)
	for _, conversation := range conversations {
		geminiRequest.Contents = append(geminiRequest.Contents, models.GeminiRequestContent{
			Role: conversation.Sender,
			Parts: append(make([]models.GeminiRequestParts, 0, 1), models.GeminiRequestParts{
				Text: conversation.Message,
			}),
		})
	}
	geminiRequest.Contents = append(geminiRequest.Contents, models.GeminiRequestContent{
		Role: "User",
		Parts: append(make([]models.GeminiRequestParts, 0, 1), models.GeminiRequestParts{
			Text: prompt,
		}),
	})
	return geminiRequest
}
func promptHandler(response http.ResponseWriter, request *http.Request, userId string) {
	prompt := request.FormValue("prompt")
	prompt = strings.Trim(prompt, "")
	if prompt == "" {
		response.WriteHeader(400)
		return
	}
	sessionId := getSessionId(userId, prompt)
	if sessionId == 0 {
		response.WriteHeader(500)
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")
	components.UserMessage(prompt, true).Render(request.Context(), response)

	geminiRequest := generateGeminiRequest(userId, sessionId, prompt)
	geminiAPIChannel := make(chan string)
	go callGeminiWithStreaming(geminiRequest, geminiAPIChannel)
	components.GeminiMessageTemplate(rand.Int()).Render(request.Context(), response)

	insertConversationChannel := make(chan int64, 2)
	defer close(insertConversationChannel)
	InsertChatConversation(sessionId, prompt, "user", insertConversationChannel)

	consolidateGeminiResponse := ""
	for message := range geminiAPIChannel {
		sendMessageAndFlush(message, response)
		consolidateGeminiResponse += message
		time.Sleep(10 * time.Millisecond)
	}

	InsertChatConversation(sessionId, consolidateGeminiResponse, "model", insertConversationChannel)
	sendMessageAndFlush("data:END\n\n", response)
	<-insertConversationChannel
	<-insertConversationChannel
}

func sendMessageAndFlush(message string, response http.ResponseWriter) {
	fmt.Fprintf(response, "%v", message)
	if flusher, ok := response.(http.Flusher); ok {
		flusher.Flush()
	}
}
func callGeminiWithStreaming(request models.GeminiRequest, channel chan<- string) {
	defer close(channel)
	url := os.Getenv("GEMINI_STREAMING_URL") + os.Getenv("GEMINI_KEY")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API %v\n", err)
		channel <- ""
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error calling Gemini API %v\n", err)
		channel <- ""
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Gemini API call: %v+\n", resp.Status)
		} else {
			fmt.Printf("Error in Gemini API call: %v+\n", string(errorMsg))
		}
		channel <- ""
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	txt := ""
	for scanner.Scan() {
		var responseParsed models.GeminiResponse
		line := scanner.Text()
		txtInLoop := line
		if strings.HasPrefix(line, "data: ") {
			txtInLoop = strings.TrimPrefix(line, "data: ")
		}
		txt += txtInLoop
		err = json.Unmarshal([]byte(txt), &responseParsed)
		if err == nil {
			channel <- responseParsed.Candidates[0].Content.Parts[0].Text
			txt = ""

		}
	}

}
func callGeminiWithoutStreaming(request models.GeminiRequest, channel chan<- string) {
	defer close(channel)
	url := os.Getenv("GEMINI_URL") + os.Getenv("GEMINI_KEY")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API %v\n", err)
		channel <- ""
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error calling Gemini API %v\n", err)
		channel <- ""
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Gemini API call: %v+\n", resp.Status)
		} else {
			fmt.Printf("Error in Gemini API call: %v+\n", string(errorMsg))
		}
		channel <- ""
		return
	}
	var responseParsed models.GeminiResponse
	err = json.NewDecoder(resp.Body).Decode(&responseParsed)
	if err != nil {
		fmt.Printf("Error parsing JSON response from Gemini API %v\n", err)
		channel <- ""
		return
	} else {
		channel <- responseParsed.Candidates[0].Content.Parts[0].Text
	}
}
