package main

import (
	"datastar-placestovisit/components"
	"datastar-placestovisit/models"
	"datastar-placestovisit/services"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

func searchCityStateCountry(responseWriter http.ResponseWriter, request *http.Request) {
	srchTxt := strings.TrimSpace(request.FormValue("srchTxt"))
	SSE := datastar.NewSSE(responseWriter, request)
	channel := make(chan []models.WorldCities)
	go services.SearchWorldCity(srchTxt, channel)
	data := <-channel
	fmt.Printf("%v search param, results count:%v\n", srchTxt, len(data))
	SSE.PatchElementTempl(components.PlacesSearchResults(data))
	SSE.PatchSignals([]byte("{showSearchResults:true}"))
}

func initializeMap(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript("if(!map){var map = L.map('map').setView([40.7834, -73.9662], 12);}", datastar.WithExecuteScriptAutoRemove(true))
	sse.ExecuteScript(`L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
		}).addTo(map);`, datastar.WithExecuteScriptAutoRemove(true))
	// var markers map[string][]string = map[string][]string{
	// 	"Central Park":                       {"40.7851", "-73.9683"},
	// 	"The Metropolitan Museum of Art":     {"40.7794", "-73.9632"},
	// 	"American Museum of Natural History": {"40.7822", "-73.9731"},
	// 	"The Guggenheim Museum":              {"40.7829", "-73.9599"},
	// 	"Strawberry Fields":                  {"40.7754", "-73.9744"},
	// }
	// index := 0
	// markerAndPopupScript := ""
	// for markerKey, markerValues := range markers {
	// 	markerAndPopupScript += `const marker` + strconv.Itoa(index) + `=L.marker([` + markerValues[0] + `,` + markerValues[1] + `]).addTo(map);`
	// 	markerAndPopupScript += `marker` + strconv.Itoa(index) + `.bindPopup("<b>` + markerKey + `</b>");`
	// 	index++
	// }
	// sse.ExecuteScript(markerAndPopupScript, datastar.WithExecuteScriptAutoRemove(true))
}
func setMap(responseWriter http.ResponseWriter, request *http.Request) {
	lat := strings.TrimSpace(chi.URLParam(request, "lat"))
	lng := strings.TrimSpace(chi.URLParam(request, "lng"))
	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript("if(map){map.setView(["+lat+","+lng+"], 12);}", datastar.WithExecuteScriptAutoRemove(true))
}
