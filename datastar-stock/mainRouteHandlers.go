package main

import (
	"datastar-stock/components"
	"datastar-stock/models"
	"datastar-stock/services"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar/sdk/go"
)

func loginHandler(responseWriter http.ResponseWriter, request *http.Request) {
	email := strings.Trim(request.FormValue("email"), "")
	password := strings.Trim(request.FormValue("password"), "")
	redirect := strings.Trim(request.FormValue("redirect"), "")
	if redirect == "" {
		redirect = "/home/populars"
	}
	signInResponse := models.SignInResponse{}
	if email != "" && password != "" {
		channel := make(chan models.SignInResponse)
		defer close(channel)
		go services.SignInEmailPassword(email, password, channel)
		signInResponse = <-channel
	}

	if email == "" || password == "" || signInResponse.ErrorMessage != "" {
		sse := datastar.NewSSE(responseWriter, request)
		sse.MergeSignals([]byte("{errorMessage:'Error. Invalid Credentials',signingIn:false}"))
	} else {
		expiresIn := time.Now().Add(55 * time.Minute) // Default to 1 hour
		expiresInParsed, err := strconv.Atoi(signInResponse.ExpiresIn)
		if err == nil {
			expiresIn = time.Now().Add(time.Duration(expiresInParsed-120) * time.Second) // add expiry 2 mins lesser , expiresin is seconds
		}

		http.SetCookie(responseWriter, &http.Cookie{
			Name:     "token",
			Value:    signInResponse.IDToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   !(os.Getenv("ENVIRONMENT") == "Development"),
			Expires:  expiresIn,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(responseWriter, request, redirect, http.StatusFound)
		return
	}
}
func tickerDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.Trim(chi.URLParam(request, "ticker"), "")
	if ticker == "" {
		http.Error(responseWriter, "Ticker not provided", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(responseWriter, request)
	cachedDataTodayChannel := make(chan []models.CacheData)
	defer close(cachedDataTodayChannel)

	go services.GetCachedData(ticker, time.Now().Format("2006-01-02"), cachedDataTodayChannel)
	chartData := <-cachedDataTodayChannel

	if (len(chartData)) == 0 {
		alphavantageChannel := make(chan models.AlphavantageResponse)
		defer close(alphavantageChannel)
		go services.CallAlphavantageAPI(ticker, alphavantageChannel)
		apiData := <-alphavantageChannel

		transformChannel := make(chan []models.CacheData)
		defer close(transformChannel)
		go transformAlphavantageResponseToCacheData(apiData, transformChannel)
		chartData = <-transformChannel

		if len(chartData) == 0 { //error
			cachedDataPrevDayChannel := make(chan []models.CacheData)
			defer close(cachedDataPrevDayChannel)
			go services.GetCachedData(ticker, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), cachedDataPrevDayChannel)
			chartData = <-cachedDataPrevDayChannel
			if len(chartData) == 0 { //still no data
				sse.MergeFragmentTempl(components.CardError(ticker))
				return
			}
		} else {
			setCacheChannel := make(chan string)
			defer close(setCacheChannel)
			go services.SetCachedData(ticker, time.Now().Format("2006-01-02"), chartData, setCacheChannel)
			<-setCacheChannel
		}
	}

	eChartDataChannel := make(chan models.EChartData)
	defer close(eChartDataChannel)
	go getEchartData(chartData, eChartDataChannel)
	eChartData := <-eChartDataChannel

	str := `LoadChart("chart_` + ticker + `",[` + eChartData.AxisData + `],[` + eChartData.ChartData + `])`
	sse.ExecuteScript(str, datastar.WithExecuteScriptAutoRemove(true))
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

func getEchartData(data []models.CacheData, channel chan<- models.EChartData) {
	eChartData := models.EChartData{}

	for idx, value := range data {
		if idx == len(data)-1 {
			eChartData.AxisData += `'` + value.Date + `'`
			eChartData.ChartData += value.Close
		} else {
			eChartData.AxisData += `'` + value.Date + `'` + ","
			eChartData.ChartData += value.Close + ","
		}
	}
	channel <- eChartData
}

func popularsDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	popularsChannel := make(chan models.PopularsFromDb)
	defer close(popularsChannel)
	go services.GetPopulars(request.Context(), popularsChannel)
	populars := <-popularsChannel

	component := components.Populars(populars.Data)
	component.Render(request.Context(), responseWriter)
}

func recentDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	recentsChannel := make(chan []models.RecentFromDb)
	defer close(recentsChannel)
	go services.GetRecent(request.Context(), recentsChannel)
	recents := <-recentsChannel

	recentsToSend := make([]string, 5)
	for idx, item := range recents {
		if idx > 4 {
			break
		}
		if strings.Trim(item.Ticker, " ") != "" {
			recentsToSend[idx] = strings.Trim(item.Ticker, " ")
		} else if strings.Trim(item.TickerLowerCase, " ") != "" {
			recentsToSend[idx] = strings.Trim(item.TickerLowerCase, " ")
		}
	}
	component := components.Recent(recentsToSend)
	component.Render(request.Context(), responseWriter)
}
