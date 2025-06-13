package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var imgRegex = regexp.MustCompile(`^data:(image/(gif|png|jpeg|jpg|webp));base64,([A-Za-z0-9+/=]+)$`)

// ALGO
// Step 1: Validate data, e.g empty prompt, invalid chatSessionId, invalid imagedata etc.
// Step 2: Insert new chat session or get all chat conversations
// Step 3: Convert chat conversation + prompt + image to GeminiRequest & call Gemini API
// Step 4: Insert user message in chat conversation & send to client
// Step 5: If new chat session inserted, send new session UI
// Step 6: If first message, update chat session title with prompt & send to client
// Step 7: Insert Gemini message in chat conversation
// Step 8: Send Gemini messages to client
// Step 9: Consolidate & Update Gemini message in chat conversation

func PromptHandler(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	userId, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	prompt := request.FormValue("prompt")
	imgBase64 := request.FormValue("imgBase64")
	chatSessionIdStr := request.FormValue("chatSessionId")
	chatSessionId, err := strconv.Atoi(chatSessionIdStr)
	newChatSessionInserted := false
	if err != nil {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	prompt = strings.TrimSpace(prompt)
	imgBase64 = strings.TrimSpace(imgBase64)
	if prompt == "" {
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	if chatSessionId == 0 {
		chatSessionId = InsertChatSessionViaChannel(userId, prompt)
		newChatSessionInserted = true
	} else {
		allChatSessions := GetChatSessionsViaChannel(userId)
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
	geminiRequest, errStr := GenerateGeminiRequest(userId, chatSessionId, prompt, imgBase64)
	if errStr != "" {
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}
	geminiAPIChannel := make(chan string)
	go callGeminiWithStreaming(geminiRequest, geminiAPIChannel)

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	insertUserChatConversationChannel := make(chan int)
	defer close(insertUserChatConversationChannel)
	go InsertChatConversation(chatSessionId, prompt, imgBase64, "user", insertUserChatConversationChannel)
	userMessageId := <-insertUserChatConversationChannel

	if userMessageId == 0 {
		sendMessageAndFlush("data:ERROR\n\n", response)
		return
	}
	components.UserMessageTemplate(userMessageId).Render(ctx, response)
	flushResponse(response)
	time.Sleep(200 * time.Millisecond)
	// sendMessageAndFlush(prompt, response)
	// time.Sleep(200 * time.Millisecond)

	if newChatSessionInserted {
		// send new session UI
		components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}, true).Render(ctx, response)
		flushResponse(response)
		time.Sleep(200 * time.Millisecond)
		components.ChatSessionIdInput(chatSessionId, false).Render(ctx, response)
		flushResponse(response)
		time.Sleep(200 * time.Millisecond)
	} else if len(geminiRequest.Contents) == 1 {
		//  update title
		chatSessionTitleChannel := make(chan int)
		defer close(chatSessionTitleChannel)
		go UpdateChatSessionTitle(userId, chatSessionId, prompt, chatSessionTitleChannel)
		rowsAffectedTitleUpdate := <-chatSessionTitleChannel
		if rowsAffectedTitleUpdate > 0 {
			components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}, false).Render(ctx, response)
			flushResponse(response)
			time.Sleep(200 * time.Millisecond)
		}
	}

	consolidateGeminiResponse := ""
	insertGeminiMessageChatConversationChannel := make(chan int)
	defer close(insertGeminiMessageChatConversationChannel)
	go InsertChatConversation(chatSessionId, consolidateGeminiResponse, "", "model", insertGeminiMessageChatConversationChannel)
	geminiMessageId := <-insertGeminiMessageChatConversationChannel
	if geminiMessageId == 0 {
		sendMessageAndFlush("data:ERROR\n\n", response)
		return
	}
	components.GeminiMessageTemplate(geminiMessageId).Render(ctx, response)
	flushResponse(response)
	time.Sleep(200 * time.Millisecond)

	for message := range geminiAPIChannel {
		if message != "data:ERROR\n\n" {
			consolidateGeminiResponse += message
		}
		select {
		case <-ctx.Done():
			continue
		default:
			sendMessageAndFlush(message, response)
			time.Sleep(100 * time.Millisecond)
		}
	}
	if strings.TrimSpace(consolidateGeminiResponse) != "" {
		updateChatConversationChannel := make(chan int)
		defer close(updateChatConversationChannel)
		go UpateGeminiMessageChatConversation(geminiMessageId, consolidateGeminiResponse, updateChatConversationChannel)
		rowsAffectedUpdate := <-updateChatConversationChannel
		if rowsAffectedUpdate == 0 {
			sendMessageAndFlush("data:ERROR\n\n", response)
			deleteChatConversationChannel := make(chan int)
			defer close(deleteChatConversationChannel)
			go DeleteGeminiMessageChatConversation(geminiMessageId, deleteChatConversationChannel)
			<-deleteChatConversationChannel
			return
		}
	}

	select {
	case <-ctx.Done():
		break
	default:
		sendMessageAndFlush("data:END\n\n", response)
		break
	}

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
	//os.WriteFile("test2.txt", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error calling Gemini API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Gemini API call: %v\n", resp.Status)
		} else {
			fmt.Printf("Error in Gemini API call: %v\n", string(errorMsg))
		}
		channel <- "data:ERROR\n\n"
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
			channel <- *responseParsed.Candidates[0].Content.Parts[0].Text
			txt = ""

		}
	}

}
