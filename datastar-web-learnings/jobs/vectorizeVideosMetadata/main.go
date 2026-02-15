package main

import (
	"context"
	"datastar-web-learnings/models"
	"datastar-web-learnings/services"
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Printf("Start Vectorizing videos...%v\n", time.Now())
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	getAllVideosChannel := make(chan []models.VideoResponse)
	go services.GetAllVideos(context.Background(), getAllVideosChannel)
	dbData := <-getAllVideosChannel
	close(getAllVideosChannel)

	// dbData = dbData[:5] // Limiting to 5 videos for testing
	fmt.Printf("Total Records to Vectorize: %v\n", len(dbData))

	videoIdToDbDataMap := make(map[string]models.VideoResponse, len(dbData))
	voyageAPIChannels := make([]chan models.VoyageEmbeddingResponse, len(dbData))
	pineconeUpsertChannels := make([]chan int, len(dbData))

	ytAPIDescriptionsChannel := make(chan models.YTAPIVideoIdAndDescription, len(dbData))
	for _, videoData := range dbData {
		videoIdToDbDataMap[videoData.VideoId] = videoData
		go callYTAPI(videoData.VideoId, ytAPIDescriptionsChannel)
	}
	ytDescriptionDataReceivedCount := 0
	for ytAPIChannelData := range ytAPIDescriptionsChannel {
		structInLoop := videoIdToDbDataMap[ytAPIChannelData.VideoId]
		structInLoop = services.ConstructTextToVectorize(structInLoop, ytAPIChannelData.Description)
		videoIdToDbDataMap[ytAPIChannelData.VideoId] = structInLoop
		dbDataIndex := 0
		for idx, data := range dbData {
			if data.VideoId == ytAPIChannelData.VideoId {
				dbDataIndex = idx
				break
			}
		}
		voyageAPIChannels[dbDataIndex] = make(chan models.VoyageEmbeddingResponse)
		go services.CallVoyageEmbedding(models.VoyageEmbeddingRequest{Input: []string{structInLoop.TextToVectorize}}, voyageAPIChannels[dbDataIndex])

		ytDescriptionDataReceivedCount += 1
		if ytDescriptionDataReceivedCount == len(dbData) {
			break
		}
	}
	close(ytAPIDescriptionsChannel)
	for idx := range dbData {
		vectorResponse := <-voyageAPIChannels[idx]
		close(voyageAPIChannels[idx])
		pineconeUpsertChannels[idx] = make(chan int)
		go services.UpsertPineconeDb(dbData[idx].VideoId, vectorResponse.Data[0].Embedding, pineconeUpsertChannels[idx])
	}
	for idx := range dbData {
		<-pineconeUpsertChannels[idx]
		close(pineconeUpsertChannels[idx])
	}
	fmt.Printf("Finished Vectorizing videos...%v\n", time.Now())
}

func callYTAPI(videoId string, ytAPIChannel chan models.YTAPIVideoIdAndDescription) {
	ytResponseChannel := make(chan models.YoutubeVideoSearchResponse)
	defer close(ytResponseChannel)
	go services.GetYTVideoResponse(videoId, ytResponseChannel)
	ytResponse := <-ytResponseChannel
	ytAPIChannel <- models.YTAPIVideoIdAndDescription{
		VideoId:     videoId,
		Description: ytResponse.Items[0].Snippet.Description,
	}
}
