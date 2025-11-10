package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
)

func QueryPineconeDb(vector []float32, channel chan<- []string) {
	resp := []string{}
	key := os.Getenv("PINECONE_API_KEY")
	hostUrl := os.Getenv("PINECONE_HOST_URL")
	url := fmt.Sprintf("%v/query", hostUrl)
	topKStr := os.Getenv("PINECONE_TOPK")
	topK, err := strconv.Atoi(topKStr)
	if err != nil {
		topK = 12
	}
	apiVersion := os.Getenv("PINECONE_API_VERSION")
	pineConeRequestBody := models.PineconeQueryRequest{
		Vector:          vector,
		TopK:            topK,
		IncludeValues:   false,
		IncludeMetadata: true,
	}
	pineConeRequestBodyStr, err := json.Marshal(pineConeRequestBody)
	if err != nil {
		fmt.Printf("Error marshalling Pinecone query request body: %v\n", err)
		channel <- resp
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(pineConeRequestBodyStr))
	if err != nil {
		fmt.Printf("Error creating HTTP request for Pinecone query: %v\n", err)
		channel <- resp
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Api-Key", key)
	httpRequest.Header.Set("X-Pinecone-API-Version", apiVersion)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for Pinecone query: %v\n", err)
		channel <- resp
		return
	}
	defer response.Body.Close()
	responseBodyRaw, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body for Pinecone query request: %v\n", err)
		channel <- resp
		return
	}
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling Pinecone query: %v\n", response.StatusCode)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	var pineconeResponse models.PineconeQueryResponse
	err = json.Unmarshal(responseBodyRaw, &pineconeResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for Pinecone query request: %v\n", err)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	if len(pineconeResponse.Matches) == 0 {
		fmt.Printf("No matches found in Pinecone query response, response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	sort.Slice(pineconeResponse.Matches, func(i, j int) bool {
		return pineconeResponse.Matches[i].Score > pineconeResponse.Matches[j].Score
	})
	for _, match := range pineconeResponse.Matches {
		// fmt.Printf("Score %v\n", match.Score)
		resp = append(resp, match.ID)
	}
	// fmt.Printf("Pinecone response IDs: %v\n", resp)
	channel <- resp
}

func UpsertPineconeDb(videoId string, vector []float32, channel chan<- int) {
	resp := 0
	key := os.Getenv("PINECONE_API_KEY")
	hostUrl := os.Getenv("PINECONE_HOST_URL")
	url := fmt.Sprintf("%v/vectors/upsert", hostUrl)

	apiVersion := os.Getenv("PINECONE_API_VERSION")
	pineConeRequestBody := models.PineconeUpsertRequest{
		Vectors: []struct {
			Id     string    `json:"id"`
			Values []float32 `json:"values"`
		}{
			{Id: videoId, Values: vector},
		},
	}
	pineConeRequestBodyStr, err := json.Marshal(pineConeRequestBody)
	if err != nil {
		fmt.Printf("Error marshalling Pinecone upsert request body: %v\n", err)
		channel <- resp
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(pineConeRequestBodyStr))
	if err != nil {
		fmt.Printf("Error creating HTTP request for Pinecone upsert: %v\n", err)
		channel <- resp
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Api-Key", key)
	httpRequest.Header.Set("X-Pinecone-API-Version", apiVersion)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for Pinecone upsert: %v\n", err)
		channel <- resp
		return
	}
	defer response.Body.Close()
	responseBodyRaw, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body for Pinecone upsert request: %v\n", err)
		channel <- resp
		return
	}
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling Pinecone upsert: %v\n", response.StatusCode)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	var pineconeResponse models.PineconeUpsertResponse
	err = json.Unmarshal(responseBodyRaw, &pineconeResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for Pinecone upsert request: %v\n", err)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	if pineconeResponse.UpsertedCount == 0 {
		fmt.Printf("No vectors upserted in Pinecone upsert response, response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	// fmt.Printf("Pinecone response : %v\n", string(responseBodyRaw))
	channel <- pineconeResponse.UpsertedCount
}
func DeleteRecordPineconeDb(videoId string, channel chan<- bool) {

	key := os.Getenv("PINECONE_API_KEY")
	hostUrl := os.Getenv("PINECONE_HOST_URL")
	url := fmt.Sprintf("%v/vectors/delete", hostUrl)

	apiVersion := os.Getenv("PINECONE_API_VERSION")
	pineConeRequestBody := map[string]interface{}{
		"ids": []string{videoId + "-1"},
	}
	pineConeRequestBodyStr, err := json.Marshal(pineConeRequestBody)
	if err != nil {
		fmt.Printf("Error marshalling Pinecone delete request body: %v\n", err)
		channel <- false
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(pineConeRequestBodyStr))
	if err != nil {
		fmt.Printf("Error creating HTTP request for Pinecone delete: %v\n", err)
		channel <- false
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Api-Key", key)
	httpRequest.Header.Set("X-Pinecone-API-Version", apiVersion)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for Pinecone delete: %v\n", err)
		channel <- false
		return
	}
	defer response.Body.Close()
	responseBodyRaw, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body for Pinecone delete request: %v\n", err)
		channel <- false
		return
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		fmt.Printf("Non 200 & 204 respone received calling Pinecone delete: %v\n", response.StatusCode)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- false
		return
	}

	// fmt.Printf("Pinecone response : %v %v\n", string(responseBodyRaw), response.StatusCode)
	channel <- true
}
