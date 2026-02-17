package main

import (
	"fmt"
	"google-drive-content-search/models"
	"google-drive-content-search/services"
	"os"
	"strconv"
	"time"
)

func main() {
	services.LoadEnv()
	fmt.Printf("Sync Process Started... %v\n", time.Now())

	deleteAllDataChannel := make(chan int)
	go services.DeleteAllData(deleteAllDataChannel)
	totalRecordsDeleted := <-deleteAllDataChannel
	close(deleteAllDataChannel)
	fmt.Printf("Cleanup: Total Records Deleted from DB: %v\n", totalRecordsDeleted)

	filesDataChannel := make(chan models.FileData)
	go services.GetFilesFromDriveV2(filesDataChannel)

	totalFilesReceived := 0
	totalRecordsInsertedToDb := 0

	for fileData := range filesDataChannel {
		totalFilesReceived++
		chunksToInsertToDb := services.ConvertFileDataCollectionToDocumentChunkCollection([]models.FileData{fileData})
		voyageRequest := models.VoyageEmbeddingRequest{}
		for _, data := range chunksToInsertToDb {
			cleanedText := services.CleanTextForVoyage(data.ChunkContent)
			voyageRequest.Input = append(voyageRequest.Input, cleanedText)
		}
		voyageResponseChannel := make(chan models.VoyageEmbeddingResponse)
		go services.CallVoyageEmbedding(voyageRequest, voyageResponseChannel)
		voyageEmbeddingResponse := <-voyageResponseChannel
		close(voyageResponseChannel)
		for _, embeddingsObj := range voyageEmbeddingResponse.Data {
			chunksToInsertToDb[embeddingsObj.Index].ChunkEmbedding = embeddingsObj.Embedding
		}
		for _, dataInCurrentLoop := range chunksToInsertToDb {
			insertChannel := make(chan int)
			go services.InsertData(dataInCurrentLoop, insertChannel)
			totalRecordsInsertedToDb += <-insertChannel
			close(insertChannel)
		}

	}

	fmt.Printf("Total Files Fetched from Drive %v\n", totalFilesReceived)
	fmt.Printf("Total Records Inserted: %v\n", totalRecordsInsertedToDb)
	fmt.Printf("Sync Process Completed. %v\n", time.Now())

}

func mainV1() {
	services.LoadEnv()
	fmt.Printf("Sync Process Started... %v\n", time.Now())

	filesDataChannel := make(chan []models.FileData)
	go services.GetFilesFromDriveV1(filesDataChannel)
	filesData := <-filesDataChannel
	close(filesDataChannel)
	fmt.Printf("Total Files Fetched from Drive & markdown generated: %v\n", len(filesData))

	dbDataToInsert := services.ConvertFileDataCollectionToDocumentChunkCollection(filesData)
	fmt.Printf("Total Document Chunks to be inserted: %v\n", len(dbDataToInsert))

	deleteAllDataChannel := make(chan int)
	go services.DeleteAllData(deleteAllDataChannel)

	voyageRequestLimit, _ := strconv.Atoi(os.Getenv("VOYAGE_REQUEST_LIMIT"))
	if voyageRequestLimit == 0 {
		voyageRequestLimit = 10
	}
	noOfRequest := (len(dbDataToInsert) + voyageRequestLimit - 1) / voyageRequestLimit
	voyageRequestChannels := make([]chan models.VoyageEmbeddingResponse, noOfRequest)
	for idx := 0; idx < noOfRequest; idx++ {
		startIndex := idx * voyageRequestLimit
		endIndex := (idx + 1) * voyageRequestLimit
		if endIndex > len(dbDataToInsert) {
			endIndex = len(dbDataToInsert)
		}
		requestToVoyage := models.VoyageEmbeddingRequest{}
		for _, data := range dbDataToInsert[startIndex:endIndex] {
			cleanedText := services.CleanTextForVoyage(data.ChunkContent)
			requestToVoyage.Input = append(requestToVoyage.Input, cleanedText)
		}
		voyageRequestChannel := make(chan models.VoyageEmbeddingResponse)
		voyageRequestChannels[idx] = voyageRequestChannel
		go services.CallVoyageEmbedding(requestToVoyage, voyageRequestChannel)
	}

	totalRecordsDeleted := <-deleteAllDataChannel
	close(deleteAllDataChannel)
	fmt.Printf("Cleanup: Total Records Deleted from DB: %v\n", totalRecordsDeleted)

	totalInserted := 0
	fileNameSet := make(map[string]bool)
	for idx := 0; idx < noOfRequest; idx++ {
		voyageEmbeddingResponse := <-voyageRequestChannels[idx]
		close(voyageRequestChannels[idx])
		startIndex := idx * voyageRequestLimit
		for _, embeddingsObj := range voyageEmbeddingResponse.Data {
			dataInCurrentLoop := dbDataToInsert[startIndex+embeddingsObj.Index]
			dataInCurrentLoop.ChunkEmbedding =
				embeddingsObj.Embedding
			if _, exists := fileNameSet[dataInCurrentLoop.FileName]; !exists {
				// fmt.Printf("Deleting existing records for file: %v\n", dataInCurrentLoop.FileName)
				deleteChannel := make(chan int)
				go services.DeleteData(dataInCurrentLoop.FileName, deleteChannel)
				<-deleteChannel
				close(deleteChannel)
				fileNameSet[dataInCurrentLoop.FileName] = true
			}
			insertChannel := make(chan int)
			// fmt.Printf("Inserting chunk index %v for file: %v\n", embeddingsObj.Index, dataInCurrentLoop.FileName)
			go services.InsertData(dataInCurrentLoop, insertChannel)
			totalInserted += <-insertChannel
			close(insertChannel)
		}
	}
	fmt.Printf("Total Records Inserted: %v\n", totalInserted)
	fmt.Printf("Sync Process Completed. %v\n", time.Now())
}
