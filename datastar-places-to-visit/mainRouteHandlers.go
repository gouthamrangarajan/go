package main

import (
	"bufio"
	"bytes"
	"datastar-placestovisit/components"
	"datastar-placestovisit/models"
	"datastar-placestovisit/services"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func searchCityStateCountry(responseWriter http.ResponseWriter, request *http.Request) {
	srchTxt := strings.TrimSpace(request.FormValue("srchTxt"))
	SSE := datastar.NewSSE(responseWriter, request)
	channel := make(chan []models.WorldCities)
	go services.SearchWorldCity(srchTxt, channel)
	data := <-channel
	SSE.PatchElementTempl(components.PlacesSearchResults(data))
	SSE.PatchSignals([]byte("{showSearchResults:true}"))
}

func initializeMap(responseWriter http.ResponseWriter, request *http.Request) {
	defaultCity := os.Getenv("DEFAULT_CITY")
	if defaultCity == "" {
		defaultCity = "Manhattan"
	}
	defaultLtd := os.Getenv("DEFAULT_LAT")
	if defaultLtd == "" {
		defaultLtd = "40.7834"
	}
	defaultLng := os.Getenv("DEFAULT_LNG")
	if defaultLng == "" {
		defaultLng = "-73.9662"
	}

	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript(`if(!map){var map = L.map('map').setView([`+defaultLtd+`,`+defaultLng+`], 12);}`, datastar.WithExecuteScriptAutoRemove(true))
	sse.ExecuteScript(`L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
		}).addTo(map);`, datastar.WithExecuteScriptAutoRemove(true))
	getPlacesSSE(sse, defaultCity, defaultLtd, defaultLng)
	sse.PatchSignals([]byte("{loadingMap:false}"))
}
func getPlaces(responseWriter http.ResponseWriter, request *http.Request) {
	city := strings.TrimSpace(chi.URLParam(request, "city"))
	lat := strings.TrimSpace(chi.URLParam(request, "lat"))
	lng := strings.TrimSpace(chi.URLParam(request, "lng"))

	if lng != "com.chrome.devtools.json" { //during debugging this value comes
		if lat == "" || lng == "" {
			http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
		} else {
			sse := datastar.NewSSE(responseWriter, request)
			getPlacesSSE(sse, city, lat, lng)
		}
	}
}
func getPlacesSSE(sse *datastar.ServerSentEventGenerator, city string, lat string, lng string) {
	dbInactivateChannel := make(chan int)
	waitForInactivate := false
	sse.PatchSignals([]byte("{loadingMap:true}"))
	sse.ExecuteScript("if(map){map.setView(["+lat+","+lng+"], 12);}", datastar.WithExecuteScriptAutoRemove(true))

	dbChannel := make(chan []models.TourismSpots)
	go services.GetSpots(lat, lng, dbChannel)
	allData := <-dbChannel
	if len(allData) > 0 {
		noOfDaysToCachePlacesInt, err := strconv.Atoi(os.Getenv("NO_OF_DAYS_TO_CACHE_SPOTS"))
		if err != nil {
			noOfDaysToCachePlacesInt = 30
		}
		noOfDaysToCachePlaces := int64(noOfDaysToCachePlacesInt)
		if time.Now().Unix()-allData[0].UnixTime > noOfDaysToCachePlaces*24*3600 {
			//data is older , inactivate & fetch new data from Gemini API
			go services.InactivateSpots(lat, lng, dbInactivateChannel)
			waitForInactivate = true

		} else {
			fmt.Printf("Using cached data for %v, %v\n", lat, lng)
			for index, data := range allData {
				sendMarkerToUI(sse, data, index+1)
			}
			sse.PatchSignals([]byte("{loadingMap:false}"))
			return
		}
	}

	geminiAiChannel := make(chan string)
	go getTourismPlacesGeminiAPI(city, lat, lng, geminiAiChannel)
	allData = []models.TourismSpots{}
	singleData := models.TourismSpots{}
	concatenatedStr := ""
	itemAlreadyExists := map[string]bool{}
	noOfPlaces, err := strconv.Atoi(os.Getenv("NO_OF_PLACES"))
	if err != nil {
		noOfPlaces = 5
	}
	for str := range geminiAiChannel {
		if str == "ERROR" {
			//handle error
		} else {
			concatenatedStr += str
			places := strings.Split(concatenatedStr, "||")
			if len(places) > 1 {
				for _, place := range places {
					if place == "" {
						continue
					}
					if len(allData) == noOfPlaces-1 && !strings.HasSuffix(place, "data:END") {
						continue
					}
					place = strings.TrimSuffix(place, "data:END")
					parts := strings.Split(place, "|")
					if len(parts) < 3 {
						continue // skip if not in expected format
					}
					singleData = models.TourismSpots{
						Name: parts[0],
						Lat:  parts[1],
						Lng:  parts[2],
					}
					key := singleData.Name + "_" + singleData.Lat + "_" + singleData.Lng
					if _, ok := itemAlreadyExists[key]; !ok {
						itemAlreadyExists[key] = true
						concatenatedStr = strings.Replace(concatenatedStr, place+"||", "", 1) // remove processed place
						allData = append(allData, singleData)
						sendMarkerToUI(sse, singleData, len(allData))
					}
				}
			} else {
				continue // wait for more data
			}
		}
	}
	sse.PatchSignals([]byte("{loadingMap:false}"))
	if waitForInactivate {
		<-dbInactivateChannel
	}
	if len(allData) > 0 {
		dbInsertChannel := make(chan string)
		go services.InsertMultipleSpot(allData, city, lat, lng, dbInsertChannel)
		<-dbInsertChannel
	}
}
func sendMarkerToUI(sse *datastar.ServerSentEventGenerator, data models.TourismSpots, markerId int) {
	markerAndPopupScript := `if (typeof marker` + strconv.Itoa(markerId) + `=='undefined') { const marker` + strconv.Itoa(markerId) + `=L.marker([` + data.Lat + `,` + data.Lng + `]).addTo(map);`
	markerAndPopupScript += `marker` + strconv.Itoa(markerId) + `.bindPopup("<b>` + data.Name + `</b>");}`
	markerAndPopupScript += `else { marker` + strconv.Itoa(markerId) + `=L.marker([` + data.Lat + `,` + data.Lng + `]).addTo(map); `
	markerAndPopupScript += `marker` + strconv.Itoa(markerId) + `.bindPopup("<b>` + data.Name + `</b>");}`
	sse.ExecuteScript(markerAndPopupScript, datastar.WithExecuteScriptAutoRemove(true))
}

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

// var markers map[string][]string = map[string][]string{
// 	"Central Park":                       {"40.7851", "-73.9683"},
// 	"The Metropolitan Museum of Art":     {"40.7794", "-73.9632"},
// 	"American Museum of Natural History": {"40.7822", "-73.9731"},
// 	// "The Guggenheim Museum":              {"40.7829", "-73.9599"},
// 	// "Strawberry Fields":                  {"40.7754", "-73.9744"},
// }
// index := 0
// markerAndPopupScript := ""
// for markerKey, markerValues := range markers {
// 	markerAndPopupScript += `let marker` + strconv.Itoa(index+1) + `=L.marker([` + markerValues[0] + `,` + markerValues[1] + `]).addTo(map);`
// 	markerAndPopupScript += `marker` + strconv.Itoa(index+1) + `.bindPopup("<b>` + markerKey + `</b>");`
// 	index++
// }
// sse.ExecuteScript(markerAndPopupScript, datastar.WithExecuteScriptAutoRemove(true))
