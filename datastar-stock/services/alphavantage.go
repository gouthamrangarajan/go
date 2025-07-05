package services

import (
	"datastar-stock/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func CallAlphavantageAPI(ticker string, channel chan<- models.AlphavantageResponse) {
	response := models.AlphavantageResponse{}
	url := os.Getenv("ALPAVANTAGE_URL") + "&symbol=" + ticker + "&apikey=" + os.Getenv("ALPAVANTAGE_API_KEY")
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error fetching data from Alphavantage:", err)
		channel <- response
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response from Alphavantage:", resp.StatusCode)
		errBody, err := io.ReadAll(resp.Body) // Read the body to avoid resource leak
		if err == nil {
			fmt.Println("Error Response from Alphavantage:", string(errBody))
		}
		channel <- response
		return
	}
	json.NewDecoder(resp.Body).Decode(&response)
	if response.MetaData.Information == "" {
		fmt.Println("Error: received empty response from Alphavantage for ticker:", ticker)
		channel <- response
		return
	}
	channel <- response
}
