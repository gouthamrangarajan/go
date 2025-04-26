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
	"strconv"
	"strings"
	"time"
)

func generateGeminiRequest(userId string, sessionId int, prompt string) models.GeminiRequest {
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
	ctx := request.Context()
	prompt := request.FormValue("prompt")
	chatSessionIdStr := request.FormValue("chatSessionId")
	chatSessionId, err := strconv.Atoi(chatSessionIdStr)
	newChatSessionInserted := false
	if err != nil {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	prompt = strings.Trim(prompt, "")
	if prompt == "" {
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	if chatSessionId == 0 {
		chatSessionId = insertChatSessionViaChannel(userId, prompt)
		newChatSessionInserted = true
	} else {
		allChatSessions := getChatSessionsViaChannel(userId)
		ftedSessions := make([]models.ChatSession, 0, 1)
		for _, session := range allChatSessions {
			if session.Id == chatSessionId {
				ftedSessions = append(ftedSessions, session)
				break
			}
		}
		if len(ftedSessions) == 0 { //RG prompt sends an chatSessionId not belonging to user
			http.Error(response, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	if chatSessionId == 0 {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	geminiRequest := generateGeminiRequest(userId, chatSessionId, prompt)
	geminiAPIChannel := make(chan string)
	go callGeminiWithStreaming(geminiRequest, geminiAPIChannel)

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")
	components.UserMessage(prompt, true).Render(ctx, response)
	flushResponse(response)
	time.Sleep(100 * time.Millisecond)

	if newChatSessionInserted {
		// send new session UI
		components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}, true).Render(ctx, response)
		flushResponse(response)
		time.Sleep(100 * time.Millisecond)
		components.ChatSessionIdInput(chatSessionId, false).Render(ctx, response)
		flushResponse(response)
		time.Sleep(100 * time.Millisecond)
	} else if len(geminiRequest.Contents) == 1 {
		//  update title
		chatSessionTitleChannel := make(chan int)
		defer close(chatSessionTitleChannel)
		go UpdateChatSessionTitle(userId, chatSessionId, prompt, chatSessionTitleChannel)
		rowsAffectedTitleUpdate := <-chatSessionTitleChannel
		if rowsAffectedTitleUpdate > 0 {
			components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}, false).Render(ctx, response)
			flushResponse(response)
			time.Sleep(100 * time.Millisecond)
		}
	}

	components.GeminiMessageTemplate(rand.Int()).Render(ctx, response)
	flushResponse(response)
	time.Sleep(100 * time.Millisecond)

	insertConversationChannel := make(chan int, 2)
	defer close(insertConversationChannel)
	InsertChatConversation(chatSessionId, prompt, "user", insertConversationChannel)

	consolidateGeminiResponse := ""

	for message := range geminiAPIChannel {
		consolidateGeminiResponse += message
		select {
		case <-ctx.Done():
			continue
		default:
			sendMessageAndFlush(message, response)
			time.Sleep(100 * time.Millisecond)
		}
	}

	InsertChatConversation(chatSessionId, consolidateGeminiResponse, "model", insertConversationChannel)

	select {
	case <-ctx.Done():
		break
	default:
		sendMessageAndFlush("data:END\n\n", response)
		break
	}

	<-insertConversationChannel
	<-insertConversationChannel
}

func sendMessageAndFlush(message string, response http.ResponseWriter) {
	fmt.Fprintf(response, "%v", message)
	flushResponse(response)
}

func flushResponse(response http.ResponseWriter) {
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
