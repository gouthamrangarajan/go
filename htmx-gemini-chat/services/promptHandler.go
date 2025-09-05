package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"htmx-gemini-chat/components"
	"htmx-gemini-chat/models"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ALGO
// Step 1:  Validate data, e.g empty prompt, invalid chatSessionId, invalid image/pdf filedata etc.
// Step 2:  Insert new chat session or get all chat conversations
// Step 3:  Convert chat conversation + prompt + image/pdf to GeminiRequest & call Gemini API / Gemini Image Generation API
// Step 4:  Image generation takes precedence and set web search flag to false for image generation
// Step 5:  Insert user message in chat conversation & send to client
// Step 6:  If new chat session inserted, send new session UI. Also call embedding to update title vector
// Step 7:  If first message, update chat session title with prompt & send to client.Also call embedding to update title vector
// step 8:  Call Gemini session web search flag update & call Image Generation Flag update
// Step 9:  Insert Gemini message in chat conversation
// Step 10: Send Gemini messages to client
// Step 11: Consolidate & Update Gemini message in chat conversation
// Step 12: If embedding called, wait for it to complete
// step 13: Wait for gemini web search flag update & Image Generation flag update to finish

func PromptHandler(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	userId, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		http.Error(response, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	prompt := request.FormValue("prompt")
	fileData := request.FormValue("base64")
	fileName := request.FormValue("fileName")
	chatSessionIdStr := request.FormValue("chatSessionId")
	chatSessionId, err := strconv.Atoi(chatSessionIdStr)

	newChatSessionInserted := false
	if err != nil {
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}
	prompt = strings.TrimSpace(prompt)
	fileData = strings.TrimSpace(fileData)

	allowWebSearchStr := request.FormValue("webSearch")
	generateImageStr := request.FormValue("imgGeneration")
	allowWebSearch := false
	generateImg := false
	allowWebSearch, _ = strconv.ParseBool(allowWebSearchStr)
	generateImg, _ = strconv.ParseBool(generateImageStr)

	if prompt == "" {
		http.Error(response, "Bad Request", http.StatusBadRequest)
		return
	}

	embeddingCallChannel := make(chan bool)
	defer close(embeddingCallChannel)
	embeddingCalled := false
	if chatSessionId == 0 {
		chatSessionId = InsertChatSessionViaChannel(userId, prompt, allowWebSearch, generateImg)
		newChatSessionInserted = true
		if chatSessionId > 0 {
			go callGeminiEmbeddingAndUpdateSessionTitleVector(chatSessionId, prompt, embeddingCallChannel)
			embeddingCalled = true
		}
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
	geminiRequest, errStr := GenerateGeminiRequest(userId, chatSessionId, prompt, fileData, allowWebSearch)
	if errStr != "" {
		http.Error(response, "Bad Request", http.StatusBadRequest)
		if embeddingCalled {
			<-embeddingCallChannel
		}
		return
	}
	geminiAPIChannel := make(chan string)
	if !generateImg {
		go callGeminiWithStreaming(geminiRequest, geminiAPIChannel)
	} else {
		allowWebSearch = false
		geminiImageRequest := models.GeminiImageGenerationRequest{
			Contents: geminiRequest.Contents,
		}
		go callGeminiImageGeneration(geminiImageRequest, geminiAPIChannel)
	}

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	insertUserChatConversationChannel := make(chan int)
	defer close(insertUserChatConversationChannel)
	go InsertChatConversation(chatSessionId, prompt, fileData, fileName, "user", insertUserChatConversationChannel)
	userMessageId := <-insertUserChatConversationChannel

	if userMessageId == 0 {
		sendMessageAndFlush("event: ERROR\n\n", response)
		if embeddingCalled {
			<-embeddingCallChannel
		}
		return
	}
	eventDataBuffer := new(bytes.Buffer)

	components.HelperText(false).Render(context.Background(), eventDataBuffer)
	sendMessageAndFlush("event: HELPER_TEXT\ndata: "+eventDataBuffer.String()+"\n\n", response)

	eventDataBuffer.Reset()
	components.UserMessageTemplate(userMessageId).Render(context.Background(), eventDataBuffer)
	sendMessageAndFlush("event: USER_MESSAGE_TEMPLATE\ndata: "+eventDataBuffer.String()+"\n\n", response)

	if newChatSessionInserted {
		// send new session UI
		eventDataBuffer.Reset()
		components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}).Render(context.Background(), eventDataBuffer)
		sendMessageAndFlush("event: MENU_ITEM\ndata: "+eventDataBuffer.String()+"\n\n", response)

		eventDataBuffer.Reset()
		components.ChatSessionIdInput(chatSessionId, false).Render(context.Background(), eventDataBuffer)
		sendMessageAndFlush("event: CHAT_SESSION_ID_INPUT\ndata: "+eventDataBuffer.String()+"\n\n", response)

	} else if len(geminiRequest.Contents) == 1 {
		//  update title
		chatSessionTitleChannel := make(chan int)
		defer close(chatSessionTitleChannel)
		go UpdateChatSessionTitle(userId, chatSessionId, prompt, chatSessionTitleChannel)
		if !embeddingCalled {
			go callGeminiEmbeddingAndUpdateSessionTitleVector(chatSessionId, prompt, embeddingCallChannel)
			embeddingCalled = true
		}
		rowsAffectedTitleUpdate := <-chatSessionTitleChannel
		if rowsAffectedTitleUpdate > 0 {
			eventDataBuffer.Reset()
			components.MenuItem(models.ChatSession{Id: chatSessionId, Title: prompt}).Render(context.Background(), eventDataBuffer)
			sendMessageAndFlush("event: MENU_ITEM\ndata: "+eventDataBuffer.String()+"\n\n", response)
		}
	}
	chatSessionWebSearchUpdateChannel := make(chan int)
	defer close(chatSessionWebSearchUpdateChannel)
	go UpdateChatSessionWebSearchFlag(userId, chatSessionId, allowWebSearch, chatSessionWebSearchUpdateChannel)

	chatSessionImgGenerationUpdateChannel := make(chan int)
	defer close(chatSessionImgGenerationUpdateChannel)
	go UpdateChatSessionImgGenerationFlag(userId, chatSessionId, generateImg, chatSessionImgGenerationUpdateChannel)

	consolidateGeminiResponse := ""
	insertGeminiMessageChatConversationChannel := make(chan int)
	defer close(insertGeminiMessageChatConversationChannel)
	go InsertChatConversation(chatSessionId, consolidateGeminiResponse, "", "", "model", insertGeminiMessageChatConversationChannel)
	geminiMessageId := <-insertGeminiMessageChatConversationChannel
	if geminiMessageId == 0 {
		sendMessageAndFlush("event: ERROR\n\n", response)
		if embeddingCalled {
			<-embeddingCallChannel
		}
		return
	}

	eventDataBuffer.Reset()
	components.GeminiMessageTemplate(geminiMessageId).Render(context.Background(), eventDataBuffer)
	sendMessageAndFlush("event: GEMINI_MESSAGE_TEMPLATE\ndata: "+eventDataBuffer.String()+"\n\n", response)

	for message := range geminiAPIChannel {
		if message != "data:ERROR\n\n" {

			consolidateGeminiResponse += message
		}
		select {
		case <-ctx.Done():
			fmt.Println("Client disconnected, stopping streaming")
			continue
		default:
			if message != "data:ERROR\n\n" {
				//not adding \n\n in the end here , might confuse find a better way
				//if added,trimend in the javscript is needed which will remove \n coming in data also
				if !generateImg {
					sendMessageAndFlush("event: MESSAGE\ndata: "+message, response)
				} else {
					sendMessageAndFlush("event: IMAGE\ndata: "+message, response)
				}
			} else {
				sendMessageAndFlush("event: ERROR\n\n", response)
			}
		}
	}
	if strings.TrimSpace(consolidateGeminiResponse) != "" {
		updateChatConversationChannel := make(chan int)
		defer close(updateChatConversationChannel)
		if !generateImg {
			go UpateGeminiMessageChatConversation(geminiMessageId, consolidateGeminiResponse, "", updateChatConversationChannel)
		} else {
			go UpateGeminiMessageChatConversation(geminiMessageId, "", consolidateGeminiResponse, updateChatConversationChannel)
		}
		rowsAffectedUpdate := <-updateChatConversationChannel
		if rowsAffectedUpdate == 0 {
			sendMessageAndFlush("event: ERROR\n\n", response)
			deleteChatConversationChannel := make(chan int)
			defer close(deleteChatConversationChannel)
			go DeleteGeminiMessageChatConversation(geminiMessageId, deleteChatConversationChannel)
			<-deleteChatConversationChannel
			if embeddingCalled {
				<-embeddingCallChannel
			}
			return
		}
	}

	select {
	case <-ctx.Done():
		fmt.Println("Client disconnected, ignoring 'event: END\n\n' message")
		break
	default:
		sendMessageAndFlush("event: END\n\n", response)
		break
	}
	if embeddingCalled {
		<-embeddingCallChannel
	}
	<-chatSessionWebSearchUpdateChannel
	<-chatSessionImgGenerationUpdateChannel
}

func sendMessageAndFlush(message string, response http.ResponseWriter) {
	fmt.Fprintf(response, "%v", message)
	if flusher, ok := response.(http.Flusher); ok {
		flusher.Flush()
	}
}

func callGeminiWithStreaming(request models.GeminiRequest, channel chan<- string) {
	defer close(channel)
	url := os.Getenv("GEMINI_STREAMING_URL")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	// fmt.Printf("Request to gemini api %v\n", string(jsonData))
	httpClient := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request for Gemini API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	httpRequest.Header.Add("x-goog-api-key", os.Getenv("GEMINI_KEY"))
	httpRequest.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(httpRequest)

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
		// fmt.Println(line)
		txtInLoop := line
		if strings.HasPrefix(line, "data: ") {
			txtInLoop = strings.TrimPrefix(line, "data: ")
		}
		txt += txtInLoop
		err = json.Unmarshal([]byte(txt), &responseParsed)
		if err == nil && len(responseParsed.Candidates) > 0 &&
			len(responseParsed.Candidates[0].Content.Parts) > 0 {
			channel <- *responseParsed.Candidates[0].Content.Parts[0].Text
			txt = ""
		}
	}

}

func callGeminiImageGeneration(request models.GeminiImageGenerationRequest, channel chan<- string) {
	defer close(channel)
	url := os.Getenv("GEMINI_IMAGE_GENERATION_URL")

	jsonData, err := json.Marshal(request)
	// os.WriteFile("test2.txt", jsonData, 0644)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini Image Generation API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	// fmt.Printf("Request to gemini image generation api %v\n", string(jsonData))

	httpClient := &http.Client{}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request for Gemini Image Generation API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	httpRequest.Header.Add("x-goog-api-key", os.Getenv("GEMINI_KEY"))
	httpRequest.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error calling Gemini Image Generation API %v\n", err)
		channel <- "data:ERROR\n\n"
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Gemini Image Generation API call: %v\n", resp.Status)
		} else {
			fmt.Printf("Error in Gemini Image Generation API call: %v\n", string(errorMsg))
		}
		channel <- "data:ERROR\n\n"
		return
	}
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body Gemini Image Generation API call%v\n", err.Error())
		channel <- "data:ERROR\n\n"
		return
	}
	// fmt.Printf("Response from Gemini Image Generation API call %v\n", string(responseBytes)[:300])
	var responseParsed models.GeminiResponse
	err = json.Unmarshal(responseBytes, &responseParsed)
	if err != nil {
		fmt.Printf("Error parsing response body Gemini Image Generation API call%v\n", err.Error())
		channel <- "data:ERROR\n\n"
		return
	}
	if len(responseParsed.Candidates) > 0 {
		for _, part := range responseParsed.Candidates[0].Content.Parts {
			if part.Text != nil && *part.Text != "" {
				fmt.Println(part.Text)
			} else if part.FileData != nil {
				imageBytes := part.FileData.Data
				channel <- "data:" + strings.TrimSpace(part.FileData.MimeType) + ";base64," + imageBytes
			}
		}
	} else {
		fmt.Printf("Response not in expected format Image Generation API call %v\n", string(responseBytes))
	}

}

func callGeminiEmbeddingAndUpdateSessionTitleVector(sessionId int, sessionTitle string, completedChannel chan bool) {
	request := GenerateGeminiEmbeddingRequest(sessionTitle)
	embeddingAPIChannel := make(chan models.GeminiEmbeddingResponse)
	defer close(embeddingAPIChannel)
	go CallGeminiEmbedding(request, embeddingAPIChannel)
	embeddingResponse := <-embeddingAPIChannel
	if len(embeddingResponse.Embedding.Values) == 0 {
		fmt.Printf("Error getting embedding response for session %d\n", sessionId)
		completedChannel <- true
		return
	}
	updateDbChannel := make(chan int)
	defer close(updateDbChannel)
	go UpdateChatSessionTitleVector(sessionId, embeddingResponse.Embedding.Values, updateDbChannel)

	<-updateDbChannel
	completedChannel <- true
}

func CallGeminiEmbedding(request models.GeminiEmbeddingRequest, channel chan<- models.GeminiEmbeddingResponse) {
	output := models.GeminiEmbeddingResponse{}
	url := os.Getenv("GEMINI_EMBEDDING_URL")
	request.Model = os.Getenv("GEMINI_EMBEDDING_MODEL")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Gemini API request %v\n", err.Error())
		channel <- output
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating http request to call Gemini API request %v\n", err.Error())
		channel <- output
		return
	}
	httpRequest.Header.Add("x-goog-api-key", os.Getenv("GEMINI_KEY"))
	httpRequest.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error calling Gemini API request %v\n", err.Error())
		channel <- output
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
		channel <- output
		return
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from Gemini API request %v\n", err.Error())
		channel <- output
		return
	}
	err = json.Unmarshal(data, &output)
	if err != nil {
		fmt.Printf("Error UnMarshalling response body from Gemini API request %v\n", err.Error())
		channel <- output
		return
	}
	channel <- output
}
