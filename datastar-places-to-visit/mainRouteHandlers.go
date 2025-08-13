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
func parseLatAndLng(lat, lng string) bool {
	if _, err := strconv.ParseFloat(lat, 32); err != nil {
		return false
	}
	if _, err := strconv.ParseFloat(lng, 32); err != nil {
		return false
	}
	return true
}
func initializeMap(responseWriter http.ResponseWriter, request *http.Request) {
	defaultVal := strings.TrimSpace(chi.URLParam(request, "default"))
	defaultCity := ""
	defaultLat := ""
	defaultLng := ""
	if defaultVal != "" {
		parts := strings.Split(defaultVal, "||")
		if len(parts) == 3 {
			defaultCity = strings.TrimSpace(parts[0])
			defaultLat = strings.TrimSpace(parts[1])
			defaultLng = strings.TrimSpace(parts[2])
		}
	}
	parseDefaultLatLngSuccess := parseLatAndLng(defaultLat, defaultLng)
	if !parseDefaultLatLngSuccess {
		defaultLat = ""
		defaultLng = ""
	}
	if defaultCity == "" && defaultLat == "" {
		defaultCity = os.Getenv("DEFAULT_CITY")
	}
	if defaultCity == "" && defaultLat == "" {
		defaultCity = "Manhattan"
	}
	if defaultLat == "" {
		defaultLat = os.Getenv("DEFAULT_LAT")
	}
	if defaultLat == "" {
		defaultLat = "40.7834"
	}
	if defaultLng == "" {
		defaultLng = os.Getenv("DEFAULT_LNG")
	}
	if defaultLng == "" {
		defaultLng = "-73.9662"
	}

	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchSignals([]byte("{loadingMap:true,selectedTab:'mapView'}"))
	sse.PatchElementTempl(components.RetryButton(defaultCity, defaultLat, defaultLng))
	if parseDefaultLatLngSuccess {
		sse.PatchElementTempl(components.SetDefaultCheckbox(true, defaultCity, defaultLat, defaultLng), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.PlacesSearchInput(defaultCity), datastar.WithUseViewTransitions(true))
	} else {
		sse.PatchElementTempl(components.SetDefaultCheckbox(false, defaultCity, defaultLat, defaultLng), datastar.WithUseViewTransitions(true))
	}
	time.Sleep(200 * time.Millisecond) //wait for tab to be available
	sse.ExecuteScript(`if(!map){var map = L.map('map').setView([`+defaultLat+`,`+defaultLng+`], 12);}`, datastar.WithExecuteScriptAutoRemove(true))
	sse.ExecuteScript(`L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
		}).addTo(map);`, datastar.WithExecuteScriptAutoRemove(true))
	getPlacesSSE(sse, defaultCity, defaultLat, defaultLng, false)

	sse.PatchSignals([]byte("{loadingMap:false}"))
}
func getPlaces(responseWriter http.ResponseWriter, request *http.Request, isRetry bool) {
	city := strings.TrimSpace(chi.URLParam(request, "city"))
	if city == "UNKNOWN" {
		city = ""
	}
	lat := strings.TrimSpace(chi.URLParam(request, "lat"))
	lng := strings.TrimSpace(chi.URLParam(request, "lng"))

	if lng != "com.chrome.devtools.json" { //during local development debugging this value comes
		if lat == "" || lng == "" {
			if request.Header.Get("datastar-request") == "true" {
				sse := datastar.NewSSE(responseWriter, request)
				showAndHideErrorMessage(sse, "Error: Invalid city selection.")
			} else {
				http.Error(responseWriter, "Bad Request", http.StatusBadRequest)
				return
			}
		} else {
			sse := datastar.NewSSE(responseWriter, request)
			sse.PatchElementTempl(components.PlacesSearchInput(city), datastar.WithUseViewTransitions(true))
			sse.PatchElementTempl(components.RetryButton(city, lat, lng))
			sse.PatchElementTempl(components.SetDefaultCheckbox(false, city, lat, lng), datastar.WithUseViewTransitions(true))
			sse.PatchSignals([]byte("{loadingMap:true,selectedTab:'mapView'}"))
			time.Sleep(200 * time.Millisecond) //wait for tab to be available
			getPlacesSSE(sse, city, lat, lng, isRetry)

		}
	}
}

func getPlacesSSE(sse *datastar.ServerSentEventGenerator, city string, lat string, lng string, isRetry bool) {
	dbInactivateChannel := make(chan int)
	waitForInactivate := false
	noOfPlaces, err := strconv.Atoi(os.Getenv("NO_OF_PLACES"))
	if err != nil {
		noOfPlaces = 5
	}
	sse.ExecuteScript("if(map){map.setView(["+lat+","+lng+"], 12);}", datastar.WithExecuteScriptAutoRemove(true))

	dbChannel := make(chan []models.TourismSpots)
	go services.GetSpots(lat, lng, noOfPlaces, dbChannel)
	allData := <-dbChannel
	if len(allData) > 0 {
		noOfDaysToCachePlacesInt, err := strconv.Atoi(os.Getenv("NO_OF_DAYS_TO_CACHE_SPOTS"))
		if err != nil {
			noOfDaysToCachePlacesInt = 30
		}
		noOfDaysToCachePlaces := int64(noOfDaysToCachePlacesInt)
		if time.Now().Unix()-allData[0].UnixTime > noOfDaysToCachePlaces*24*3600 || isRetry {
			//data is older , inactivate & fetch new data from Gemini API
			go services.InactivateSpots(lat, lng, dbInactivateChannel)
			waitForInactivate = true

		} else {
			fmt.Printf("Using cached data for %v, %v\n", lat, lng)
			for _, data := range allData {
				sendMarkerToUI(sse, data)
				sendTableRowToUI(sse, data)
			}
			sse.PatchSignals([]byte("{loadingMap:false}"))
			return
		}
	}

	geminiAiChannel := make(chan string)
	go getTourismPlacesGeminiAPI(lat, lng, noOfPlaces, geminiAiChannel)
	if len(allData) > 0 && isRetry {
		removeTableRowAndMarkerFromUI(sse, allData)
	}
	allData = []models.TourismSpots{}
	singleData := models.TourismSpots{}
	concatenatedStr := ""
	itemAlreadyExists := map[string]bool{}

	for str := range geminiAiChannel {
		if str == "ERROR" {
			showAndHideErrorMessage(sse, "Error: Please try again later.")
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
					if len(parts) < 4 {
						continue // skip if not in expected format
					}
					singleData = models.TourismSpots{
						Name:        parts[0],
						Description: parts[1],
						Lat:         parts[2],
						Lng:         parts[3],
					}
					key := singleData.Name
					if _, ok := itemAlreadyExists[key]; !ok {
						itemAlreadyExists[key] = true
						concatenatedStr = strings.Replace(concatenatedStr, place+"||", "", 1) // remove processed place
						idChannel := make(chan int)
						go services.InsertSpot(singleData, city, lat, lng, idChannel)
						singleData.Id = <-idChannel
						close(idChannel)
						if singleData.Id > 0 {
							allData = append(allData, singleData)
							sendMarkerToUI(sse, singleData)
							sendTableRowToUI(sse, singleData)
						}
						sse.PatchSignals([]byte("{loadingMap:false}"))
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

}

func sendMarkerToUI(sse *datastar.ServerSentEventGenerator, data models.TourismSpots) {
	markerVariableName := "marker_" + strconv.Itoa(data.Id)
	markerAndPopupScript := `if (typeof ` + markerVariableName + `=='undefined') { var ` + markerVariableName + `=L.marker([` + data.Lat + `,` + data.Lng + `]).addTo(map);`
	markerAndPopupScript += markerVariableName + `.bindPopup("<b>` + data.Name + `</b>");}`
	markerAndPopupScript += `else { ` + markerVariableName + `.remove(); ` + markerVariableName + `=L.marker([` + data.Lat + `,` + data.Lng + `]).addTo(map); `
	markerAndPopupScript += markerVariableName + `.bindPopup("<b>` + data.Name + `</b>");}`
	sse.ExecuteScript(markerAndPopupScript, datastar.WithExecuteScriptAutoRemove(true))
}

func sendTableRowToUI(sse *datastar.ServerSentEventGenerator, data models.TourismSpots) {
	// sse.RemoveElement("#tr_"+strconv.Itoa(data.Id), datastar.WithUseViewTransitions(true))
	sse.ExecuteScript(`if (document.getElementById("tr_`+strconv.Itoa(data.Id)+`")) { document.getElementById("tr_`+strconv.Itoa(data.Id)+`").remove(); }`, datastar.WithExecuteScriptAutoRemove(true))
	sse.PatchElementTempl(components.PlacesTableRow(data), datastar.WithModeAppend(), datastar.WithSelectorID("tableViewTbody"), datastar.WithUseViewTransitions(true))
}

func removeTableRowAndMarkerFromUI(sse *datastar.ServerSentEventGenerator, allData []models.TourismSpots) {
	for _, record := range allData {
		markerVariableName1 := "marker_" + strconv.Itoa(record.Id)
		sse.RemoveElement("#tr_"+strconv.Itoa(record.Id), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript(`if (typeof `+markerVariableName1+`!=='undefined'){`+markerVariableName1+`.remove();}`, datastar.WithExecuteScriptAutoRemove(true))
	}
}
func showGettingCoordinatesError(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	showAndHideErrorMessage(sse, "Location access required. Please enable location services and click 'Allow' to continue.")
}
func showAndHideErrorMessage(sse *datastar.ServerSentEventGenerator, message string) {
	errorId := rand.Int()
	sse.PatchElementTempl(components.ErrorMessage(errorId, message), datastar.WithModeAppend(), datastar.WithSelectorID("errorContainer"), datastar.WithUseViewTransitions(true))
	time.Sleep(3 * time.Second)
	sse.PatchElementTempl(components.ErrorMessageAnimateOut(errorId, message))
	time.Sleep(2 * time.Millisecond)
	sse.RemoveElement("#error-"+strconv.Itoa(errorId), datastar.WithUseViewTransitions(true))
}
