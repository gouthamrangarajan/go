package main

import (
	"datastar-claude-chat/models"
	"datastar-claude-chat/services"
	"fmt"
	"os"

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
	defer close(getAllChatSessionsChannel)
	go services.GetAllChatSessionsForJob(getAllChatSessionsChannel)
	allSessions := <-getAllChatSessionsChannel
	fmt.Printf("Total sessions: %d\n", len(allSessions))
	embeddingChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(embeddingChannel)
	embeddingRequest := models.VoyageEmbeddingRequest{
		Model: os.Getenv("VOYAGE_EMBEDDINGS_MODEL"),
		Input: []string{},
	}
	dbUpdateChannels := make([]chan int, len(allSessions))
	indexKeyToIdValMapping := make(map[int]int, len(allSessions))
	requestoVectors := make([]string, len(allSessions))

	for idx, session := range allSessions {
		titleToVectorize := session.Title
		if len(titleToVectorize) > 500 {
			titleToVectorize = session.Title[:500]
		}
		requestoVectors[idx] = titleToVectorize
		indexKeyToIdValMapping[idx] = session.Id
		embeddingRequest.Input = append(embeddingRequest.Input, titleToVectorize)
	}
	go services.CallVoyageEmbedding(embeddingRequest, embeddingChannel)
	embeddingResponse := <-embeddingChannel

	dbUpdateCalled := false
	if len(embeddingResponse.Data) > 0 {
		dbUpdateCalled = true
		for idx, embeddingItem := range embeddingResponse.Data {
			dbUpdateChannels[idx] = make(chan int)
			defer close(dbUpdateChannels[idx])
			sessionIdToUpdate := indexKeyToIdValMapping[embeddingItem.Index]
			go services.UpdateChatSessionTitleVector(sessionIdToUpdate, embeddingItem.Embedding, dbUpdateChannels[idx])
		}
	}
	if dbUpdateCalled {
		for _, dbUpdateChannel := range dbUpdateChannels {
			fmt.Println("DB update result:", <-dbUpdateChannel)
		}
	}
	fmt.Println("Completed the session title vector job...")
}
