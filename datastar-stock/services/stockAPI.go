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
	} else {
		fmt.Printf("Successfully fetched data from Alphavantage API for ticker %s\n", ticker)
	}
	channel <- response
}

func CallFMPAPI(ticker string, channel chan<- []models.FMPResponse) {
	response := []models.FMPResponse{}
	url := os.Getenv("FINANCIAL_MODELLING_PREP_URL") + "?symbol=" + ticker + "&apikey=" + os.Getenv("FINANCIAL_MODELLING_PREP_API_KEY")
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error fetching data from FMP API:", err)
		channel <- response
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response from FMP API:", resp.StatusCode)
		if err == nil {
			fmt.Println("Error Response from FMP API:", string(respBody))
		}
		channel <- response
		return
	}
	json.Unmarshal(respBody, &response)
	if len(response) > 0 {
		fmt.Printf("Successfully fetched data from FMP API for ticker %s\n", ticker)
	} else {
		fmt.Printf("Invalid Response for Ticker %v from FMP API:%v\n", ticker, string(respBody))
	}
	channel <- response
}

func CallTwelveDataAPI(ticker string, channel chan<- models.TwelveDataResponse) {
	response := models.TwelveDataResponse{}
	url := os.Getenv("TWELVE_DATA_URL") + "?symbol=" + ticker + "&interval=1day" + "&apikey=" + os.Getenv("TWELVE_DATA_API_KEY")
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error fetching data from Twelve Data API:", err)
		channel <- response
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: received non-200 response from Twelve Data API:", resp.StatusCode)
		if err == nil {
			fmt.Println("Error Response from Twelve Data API:", string(respBody))
		}
		channel <- response
		return
	}
	json.Unmarshal(respBody, &response)
	if len(response.Values) > 0 {
		fmt.Printf("Successfully fetched data from Twelve Data API for ticker %s\n", ticker)
	} else {
		fmt.Printf("Invalid Response for Ticker %v from Twelve Data API:%v\n", ticker, string(respBody))
	}
	channel <- response
}
