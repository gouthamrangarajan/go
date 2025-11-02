package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		fmt.Printf("Error marshalling Pinecone request body: %v\n", err)
		channel <- resp
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(pineConeRequestBodyStr))
	if err != nil {
		fmt.Printf("Error creating HTTP request for Pinecone: %v\n", err)
		channel <- resp
		return
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Api-Key", key)
	httpRequest.Header.Set("X-Pinecone-API-Version", apiVersion)
	client := &http.Client{}
	response, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error making HTTP request for Pinecone: %v\n", err)
		channel <- resp
		return
	}
	defer response.Body.Close()
	responseBodyRaw, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error reading response body for Pinecone request: %v\n", err)
		channel <- resp
		return
	}
	if response.StatusCode != http.StatusOK {
		fmt.Printf("Non 200 respone received calling Pinecone: %v\n", response.StatusCode)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	var pineconeResponse models.PineconeQueryResponse
	err = json.Unmarshal(responseBodyRaw, &pineconeResponse)
	if err != nil {
		fmt.Printf("Error unmarshalling response body for Pinecone request: %v\n", err)
		fmt.Printf("Response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	if len(pineconeResponse.Matches) == 0 {
		fmt.Printf("No matches found in Pinecone response, response body: %v\n", string(responseBodyRaw))
		channel <- resp
		return
	}
	for _, match := range pineconeResponse.Matches {
		resp = append(resp, match.ID)
	}
	// fmt.Printf("Pinecone response IDs: %v\n", resp)
	channel <- resp
}
