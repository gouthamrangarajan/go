package main

import (
	"datastar-openrouter/models"
	"datastar-openrouter/services"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Success loaded .env file")
	}
	services.InitDB()
	fmt.Printf("Starting the session title vector job %v\n...", time.Now())
	getAllChatSessionsChannel := make(chan []models.ChatSession)
	defer close(getAllChatSessionsChannel)
	go services.GetAllChatSessionsForJob(getAllChatSessionsChannel)
	allSessions := <-getAllChatSessionsChannel
	fmt.Printf("Total sessions: %d\n", len(allSessions))

	voyageAPIRequestLimitStr := os.Getenv("VOYAGE_REQUEST_LIMIT")
	voyageAPIRequestLimit, _ := strconv.Atoi(voyageAPIRequestLimitStr)
	if voyageAPIRequestLimit == 0 {
		voyageAPIRequestLimit = 10
	}

	noOfVoyageAPICalls := (len(allSessions) + voyageAPIRequestLimit - 1) / voyageAPIRequestLimit
	fmt.Printf("Total Voyage API calls to be made: %d\n", noOfVoyageAPICalls)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(noOfVoyageAPICalls)
	for idx := 0; idx < noOfVoyageAPICalls; idx++ {
		startIdx := idx * voyageAPIRequestLimit
		endIdx := (idx + 1) * voyageAPIRequestLimit
		if endIdx > len(allSessions) {
			endIdx = len(allSessions)
		}
		sessionsToUpdateTitleVector := allSessions[startIdx:endIdx]
		go workerCallVoyageAPIAndUpdateDb(sessionsToUpdateTitleVector, &waitGroup)
	}
	waitGroup.Wait()
	fmt.Printf("Completed the session title vector job %v\n...", time.Now())
}

func workerCallVoyageAPIAndUpdateDb(dbData []models.ChatSession, wg *sync.WaitGroup) {
	defer wg.Done()
	voyageRequestChannel := make(chan models.VoyageEmbeddingResponse)
	defer close(voyageRequestChannel)
	voyageAPIRequest := models.VoyageEmbeddingRequest{
		Input: []string{},
	}
	for _, session := range dbData {
		titleToVectorize := session.Title
		if len(titleToVectorize) > 500 {
			titleToVectorize = session.Title[:500]
		}
		voyageAPIRequest.Input = append(voyageAPIRequest.Input, titleToVectorize)
	}
	go services.CallVoyageEmbedding(voyageAPIRequest, voyageRequestChannel)
	voyageAPIResponse := <-voyageRequestChannel
	if len(voyageAPIResponse.Data) > 0 {
		dbUpdateChannels := make([]chan int, len(voyageAPIResponse.Data))
		for _, embeddingItem := range voyageAPIResponse.Data {
			sessionIdToUpdate := dbData[embeddingItem.Index].Id
			// fmt.Printf("Received embedding for session id %v\n", sessionIdToUpdate)
			dbUpdateChannels[embeddingItem.Index] = make(chan int)
			go services.UpdateChatSessionTitleVector(sessionIdToUpdate, embeddingItem.Embedding, dbUpdateChannels[embeddingItem.Index])
		}
		for _, dbUpdateChannel := range dbUpdateChannels {
			result := <-dbUpdateChannel
			if result == 1 {
				fmt.Printf("Successfully updated session title vector for session id %v\n", dbData[result].Id)
			}
			close(dbUpdateChannel)
		}
	}

}
