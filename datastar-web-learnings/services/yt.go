package services

import (
	"datastar-web-learnings/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type YTService struct {
	apiUrl string
	apiKey string
}

func NewYTService() *YTService {
	return &YTService{
		apiUrl: os.Getenv("YT_API_URL"),
		apiKey: os.Getenv("YT_API_KEY"),
	}
}
func (y *YTService) GetYTVideoResponse(videoId string, channel chan<- models.YoutubeVideoSearchResponse) {
	defer close(channel)
	var ytResponse models.YoutubeVideoSearchResponse
	client := &http.Client{}
	resp, err := client.Get(y.apiUrl + `/videos?part=snippet&id=` + videoId + `&key=` + y.apiKey)
	if err != nil {
		fmt.Printf("Error making HTTP request to YT API: %v\n", err)
		channel <- ytResponse
		return
	}
	defer resp.Body.Close()
	respBodyRaw, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Non-OK HTTP status from YT API: %v\n", resp.Status)
		if err == nil {
			fmt.Printf("Response body: %v\n", string(respBodyRaw))
		}
		channel <- ytResponse
		return
	}
	if err == nil {
		err = json.Unmarshal(respBodyRaw, &ytResponse)
		if err != nil {
			fmt.Printf("Error unmarshalling YT API response: %v\n", err)
			channel <- ytResponse
			return
		}
	} else {
		fmt.Printf("Error reading YT API response body: %v\n", err)
	}
	channel <- ytResponse
}
