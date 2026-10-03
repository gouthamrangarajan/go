package voyage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Client struct {
	url    string
	model  string
	apiKey string
}

func NewClient() *Client {
	return &Client{
		url:    os.Getenv("VOYAGE_EMBEDDINGS_URL"),
		model:  os.Getenv("VOYAGE_EMBEDDINGS_MODEL"),
		apiKey: os.Getenv("VOYAGE_API_KEY"),
	}
}

func (c *Client) CallEmbedding(request Request, channel chan<- Response) {
	defer close(channel)
	output := Response{}
	url := c.url
	request.Model = c.model

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
	httpRequest.Header.Add("Authorization", `Bearer `+c.apiKey)
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
