package services

import (
	"datastar-web-learnings/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func GetYTVideoResponse(videoId string, channel chan<- models.YoutubeVideoSearchResponse) {
	var ytResponse models.YoutubeVideoSearchResponse
	client := &http.Client{}
	resp, err := client.Get(os.Getenv("YT_API_URL") + `/videos?part=snippet&id=` + videoId + `&key=` + os.Getenv("YT_API_KEY"))
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
