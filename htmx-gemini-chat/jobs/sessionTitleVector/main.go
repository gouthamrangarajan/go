package main

import (
	"fmt"
	"htmx-gemini-chat/models"
	"htmx-gemini-chat/services"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Success loaded .env file")
	}
	fmt.Println("Starting the session title vector job...")
	getAllChatSessionsChannel := make(chan []models.ChatSession)
	go services.GetAllChatSessionsForJob(getAllChatSessionsChannel)
	allSessions := <-getAllChatSessionsChannel
	fmt.Printf("Total sessions: %d\n", len(allSessions))
	geminiEmbeddingChannels := make([]chan models.GeminiEmbeddingResponse, len(allSessions))
	dbUpdateChannels := make([]chan int, len(allSessions))

	for idx, session := range allSessions {
		request := services.GenerateGeminiEmbeddingRequest(session.Title)
		geminiEmbeddingChannels[idx] = make(chan models.GeminiEmbeddingResponse)
		defer close(geminiEmbeddingChannels[idx])
		go services.CallGeminiEmbedding(request, geminiEmbeddingChannels[idx])
	}
	for idx, geminiEmbeddingChannel := range geminiEmbeddingChannels {
		session := allSessions[idx]
		resp := <-geminiEmbeddingChannel
		dbUpdateChannels[idx] = make(chan int)
		defer close(dbUpdateChannels[idx])
		go services.UpdateChatSessionTitleVector(session.Id, resp.Embedding.Values, dbUpdateChannels[idx])
	}
	for _, dbUpdateChannel := range dbUpdateChannels {
		fmt.Println("DB update result:", <-dbUpdateChannel)
	}
	fmt.Println("Completed the session title vector job...")
}
