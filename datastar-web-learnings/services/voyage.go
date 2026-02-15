package services

import (
	"bytes"
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func CallVoyageEmbedding(request models.VoyageEmbeddingRequest, channel chan<- models.VoyageEmbeddingResponse) {
	output := models.VoyageEmbeddingResponse{}
	url := os.Getenv("VOYAGE_EMBEDDINGS_URL")
	request.Model = os.Getenv("VOYAGE_EMBEDDINGS_MODEL")

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error converting request to json data to call Voyage API request %v\n", err.Error())
		channel <- output
		return
	}
	httpRequest, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating http request to call Voyage API request %v\n", err.Error())
		channel <- output
		return
	}
	httpRequest.Header.Add("Authorization", `Bearer `+os.Getenv("VOYAGE_API_KEY"))
	httpRequest.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("Error calling Voyage API request %v\n", err.Error())
		channel <- output
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errorMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in Voyage API call: %v\n", resp.Status)
		} else {
			fmt.Printf("Error in Voyage API call: %v\n", string(errorMsg))
		}
		channel <- output
		return
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from Voyage API request %v\n", err.Error())
		channel <- output
		return
	}
	err = json.Unmarshal(data, &output)
	if err != nil {
		fmt.Printf("Error UnMarshalling response body from Voyage API request %v\n", err.Error())
		channel <- output
		return
	}
	channel <- output
}
