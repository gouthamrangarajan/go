package main

import (
	"datastar-placestovisit/components"
	"datastar-placestovisit/models"
	"datastar-placestovisit/services"
	"fmt"
	"math/rand"
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
			if request.Header.Get("datastar-request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				errorId := rand.Int()
				sse.PatchElementTempl(components.ErrorMessage(errorId, "Error: Invalid city selection."), datastar.WithModeAppend(), datastar.WithSelectorID("errorContainer"), datastar.WithUseViewTransitions(true))
				time.Sleep(3 * time.Second)
				sse.RemoveElement("#error-"+strconv.Itoa(errorId), datastar.WithUseViewTransitions(true))
			} else {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
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
			errorId := rand.Int()
			sse.PatchElementTempl(components.ErrorMessage(errorId, "Error: Please try again later."), datastar.WithModeAppend(), datastar.WithSelectorID("errorContainer"), datastar.WithUseViewTransitions(true))
			time.Sleep(3 * time.Second)
			sse.RemoveElement("#error-"+strconv.Itoa(errorId), datastar.WithUseViewTransitions(true))
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

func showGettingCoordinatesError(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	errorId := rand.Int()
	sse.PatchElementTempl(components.ErrorMessage(errorId, "Error: Please enable location service & click on allow Know your location."), datastar.WithModeAppend(), datastar.WithSelectorID("errorContainer"), datastar.WithUseViewTransitions(true))
	time.Sleep(3 * time.Second)
	sse.RemoveElement("#error-"+strconv.Itoa(errorId), datastar.WithUseViewTransitions(true))
}
