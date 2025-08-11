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
	"strconv"
	"strings"
)

func getPrompt1(lat, lng string, noOfPlaces int) string {
	prompt := `You are an AI assistant specialized in providing concise tourism information for a given geographical location.
				I will provide you with a central latitude and longitude.
				Your task is to identify the **top 5 most popular or significant tourism spots** in the area defined by these coordinates.
				For each spot, provide its **name**, a **very brief description** (1-2 concise sentences), its **latitude**, and its **longitude**.

				**Important Formatting Rules:**
				1.  Each spot's details (name, description, latitude, longitude) must be separated by a single pipe: '|'.
				2.  Each complete spot entry (all its details) must be separated from the next by a double pipe: '||'.
				3.  Do NOT include any introductory or concluding text. Provide ONLY the formatted output.
				4.  Ensure latitude and longitude are provided as decimal numbers (e.g., 48.8584, 2.2945).
				5.  Do NOT include any line breaks within a single spot's entry. Only '||' should separate entries.

				**Input Example:**
				Latitude: 48.8566
				Longitude: 2.3522

				**Expected Output Format Example:**
				Eiffel Tower|Iconic iron landmark offering panoramic views of Paris.|48.8584|2.2945||Louvre Museum|World-renowned art museum, home to the Mona Lisa and countless masterpieces.|48.8606|2.3376||Notre-Dame Cathedral|Historic Gothic cathedral, famous for its architecture and gargoyles.|48.8530|2.3499||Arc de Triomphe|Neoclassical monument commemorating French victories, standing at the western end of the Champs-Élysées.|48.8738|2.2950||Champs-Élysées|Famous avenue known for its theaters, cafes, and luxury shops.|48.8698|2.3072

				**Your turn. Provide the top ` + strconv.Itoa(noOfPlaces) + ` tourism spots for the following coordinates:**

				Latitude: ` + lat + `
				Longitude: ` + lng
	return prompt
}
func getPrompt2(lat, lng string, noOfPlaces int) string {
	prompt := `You are an assistant that provides the top ` + strconv.Itoa(noOfPlaces) + ` tourism spots based on the given city and/or geographic coordinates. For each spot, provide the name, a short description (1-2 sentences), the latitude, and longitude.

				Format your response exactly as follows:
				name|description|latitude|longitude||name|description|latitude|longitude||name|description|latitude|longitude||name|description|latitude|longitude||name|description|latitude|longitude

				Make sure to provide exactly 5 items separated by "||". Use '|' to separate the name, description, latitude, and longitude within each item.

				Rely solely on the latitude and longitude coordinates.
				
				The latitude and longitude are: ` + lat + `,` + lng + `

				Only provide the formatted list, no additional text.`
	return prompt
}
func getTourismPlacesGeminiAPI(lat, lng string, noOfPlaces int, channel chan string) {
	defer close(channel)
	geminiRequest := models.GeminiRequest{
		Contents: []models.GeminiRequestContent{},
	}
	geminiRequest.Contents = append(geminiRequest.Contents, models.GeminiRequestContent{
		Role:  "user",
		Parts: []models.GeminiRequestContentPart{},
	})
	text := getPrompt1(lat, lng, noOfPlaces)
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
			if len(responseParsed.Candidates) == 0 ||
				len(responseParsed.Candidates[0].Content.Parts) == 0 {
				fmt.Printf("Unexpected format found in response: %s\n", txt)

			} else {
				channel <- *responseParsed.Candidates[0].Content.Parts[0].Text
			}
			txt = ""
		}
	}
	channel <- "data:END||"
}
