package services

import (
	"datastar-stock/models"
	"sort"
	"strconv"
)

func CallStockApisInPriority(ticker string) []models.CacheData {
	chartData := make([]models.CacheData, 0)
	transformChannel := make(chan []models.CacheData)
	defer close(transformChannel)

	fmpChannel := make(chan []models.FMPResponse)
	defer close(fmpChannel)

	go CallFMPAPI(ticker, fmpChannel)
	fmpData := <-fmpChannel
	if len(fmpData) > 0 {
		go transformFMPResponseToCacheData(fmpData, transformChannel)
		chartData = <-transformChannel
	} else {
		alphavantageChannel := make(chan models.AlphavantageResponse)
		defer close(alphavantageChannel)
		go CallAlphavantageAPI(ticker, alphavantageChannel)
		alphavantageData := <-alphavantageChannel

		go transformAlphavantageResponseToCacheData(alphavantageData, transformChannel)
		chartData = <-transformChannel
	}
	if len(chartData) == 0 {
		twelveDataApiChannel := make(chan models.TwelveDataResponse)
		defer close(twelveDataApiChannel)
		go CallTwelveDataAPI(ticker, twelveDataApiChannel)
		twelveApiData := <-twelveDataApiChannel
		go transformTwelveDataResponseToCacheData(twelveApiData, transformChannel)
		chartData = <-transformChannel
	}
	return chartData
}
func transformAlphavantageResponseToCacheData(response models.AlphavantageResponse, channel chan<- []models.CacheData) {
	chartData := make([]models.CacheData, 0)
	dates := make([]string, 0, len(response.TimeSeriesDaily))
	for date := range response.TimeSeriesDaily {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	for _, date := range dates {
		dailyData := response.TimeSeriesDaily[date]
		chartData = append(chartData, models.CacheData{
			Date:   date,
			Close:  dailyData.Close,
			Open:   dailyData.Open,
			High:   dailyData.High,
			Low:    dailyData.Low,
			Volume: dailyData.Volume,
		})
	}
	channel <- chartData
}
func transformFMPResponseToCacheData(response []models.FMPResponse, channel chan<- []models.CacheData) {
	chartData := make([]models.CacheData, 0)

	for _, data := range response {
		chartData = append(chartData, models.CacheData{
			Date:   data.Date,
			Close:  strconv.FormatFloat(data.Close, 'f', 2, 64),
			Open:   strconv.FormatFloat(data.Open, 'f', 2, 64),
			High:   strconv.FormatFloat(data.High, 'f', 2, 64),
			Low:    strconv.FormatFloat(data.Low, 'f', 2, 64),
			Volume: strconv.Itoa(data.Volume),
		})
	}
	for i, j := 0, len(chartData)-1; i < j; i, j = i+1, j-1 {
		chartData[i], chartData[j] = chartData[j], chartData[i]
	}
	channel <- chartData
}

func transformTwelveDataResponseToCacheData(response models.TwelveDataResponse, channel chan<- []models.CacheData) {
	chartData := make([]models.CacheData, 0)

	for _, data := range response.Values {
		chartData = append(chartData, models.CacheData{
			Date:   data.Date,
			Close:  data.Close,
			Open:   data.Open,
			High:   data.High,
			Low:    data.Low,
			Volume: data.Volume,
		})
	}
	for i, j := 0, len(chartData)-1; i < j; i, j = i+1, j-1 {
		chartData[i], chartData[j] = chartData[j], chartData[i]
	}
	channel <- chartData
}
