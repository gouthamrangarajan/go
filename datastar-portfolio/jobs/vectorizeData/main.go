package main

import (
	"datastar-portfolio/models"
	"datastar-portfolio/services"
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	fmt.Printf("Job Started %v\n", time.Now())
	getChannel := make(chan []models.DemoItem)
	go services.GetAllDemos(getChannel, false)
	demos := <-getChannel
	close(getChannel)
	fmt.Printf("Fetched %v demo items from the database:\n", len(demos))
	// fmt.Printf("Inserted %v demo items into the database: %v\n", insertedCount, time.Now())
	voyageRequest := models.VoyageEmbeddingRequest{}
	voyageRequest.Input = []string{}
	for _, demo := range demos {
		strToEmbed := fmt.Sprintf("Project Title: %s\n"+
			"Deployment Service: %s\n"+
			"Technologies: %s\n"+
			"Description: %s",
			demo.Title, demo.Service, demo.Tags, demo.Description)
		voyageRequest.Input = append(voyageRequest.Input, strToEmbed)
	}
	voyageChannel := make(chan models.VoyageEmbeddingResponse)
	go services.CallVoyageEmbedding(voyageRequest, voyageChannel)
	voyageResponse := <-voyageChannel
	fmt.Printf("Received embedding response from Voyage API with %v embeddings\n", len(voyageResponse.Data))
	close(voyageChannel)

	for _, embeddingItem := range voyageResponse.Data {
		demos[embeddingItem.Index].Embeddings = embeddingItem.Embedding
	}
	updateChannel := make(chan int)
	go services.UpdateDemosEmbeddings(demos, updateChannel)
	updatedCount := <-updateChannel
	fmt.Printf("Updated %v demo items with embeddings in the database: %v\n", updatedCount, time.Now())
}
