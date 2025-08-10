package main

import (
	"bufio"
	"bytes"
	"datastar-placestovisit/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func getTourismPlacesGeminiAPI(city string, lat string, lng string, channel chan string) {
	defer close(channel)
	geminiRequest := models.GeminiRequest{
		Contents: []models.GeminiRequestContent{},
	}
	geminiRequest.Contents = append(geminiRequest.Contents, models.GeminiRequestContent{
		Role:  "user",
		Parts: []models.GeminiRequestContentPart{},
	})
	text := `What are the top ` + os.Getenv("NO_OF_PLACES") + ` tourism places to visit in the world `
	if city != "" {
		text += ` in the city ` + city
	}
	text += ` at latitude ` + lat + ` and longitude ` + lng + `?`
	text += `Please provide the name, latitude, and longitude of each place.
			 Separate the name, latitude and longitude with a '|'. 
			 Separate the places with a '||'. 
			 Do not include any other information or formatting.`
	geminiRequest.Contents[0].Parts = append(geminiRequest.Contents[0].Parts, models.GeminiRequestContentPart{
		Text: &text,
	})
	url := os.Getenv("GEMINI_STREAMING_URL") + os.Getenv("GEMINI_KEY")
	jsonData, err := json.Marshal(geminiRequest)
	if err != nil {
		fmt.Printf("Error marshalling Gemini request: %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making request to Gemini API: %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Gemini API returned non-200 status code: %v\n", resp.StatusCode)
		errorMessage, err := io.ReadAll(resp.Body)
		if err == nil {
			fmt.Printf("Error message from Gemini API: %s\n", errorMessage)
		}
		channel <- "ERROR"
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	txt := ""
	for scanner.Scan() {
		var responseParsed models.GeminiResponse
		line := scanner.Text()
		txtInLoop := line
		if strings.HasPrefix(line, "data: ") {
			txtInLoop = strings.TrimPrefix(line, "data: ")
		}
		txt += txtInLoop
		err = json.Unmarshal([]byte(txt), &responseParsed)
		if err == nil {
			channel <- *responseParsed.Candidates[0].Content.Parts[0].Text
			txt = ""
		}
	}
	channel <- "data:END||"
}
