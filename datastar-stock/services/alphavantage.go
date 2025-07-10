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
	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response from Alphavantage:", resp.StatusCode)
		if err == nil {
			fmt.Println("Error Response from Alphavantage:", string(respBody))
		}
		channel <- response
		return
	}
	json.Unmarshal(respBody, &response)
	if response.MetaData.Information == "" {
		fmt.Printf("Error: received invalid response from Alphavantage for ticker %v:, response:%v\n", ticker, string(respBody))
		channel <- response
		return
	}
	channel <- response
}
