package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"datastar-stock/components"
	"datastar-stock/models"
	"datastar-stock/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	datastar "github.com/starfederation/datastar/sdk/go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Compress(5))
	router.Use(services.LoggedIn)
	router.Get("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		component := components.Landing()
		component.Render(request.Context(), responseWriter)
	})
	router.Post("/login", func(responseWriter http.ResponseWriter, request *http.Request) {
		email := strings.Trim(request.FormValue("email"), "")
		password := strings.Trim(request.FormValue("password"), "")
		redirect := strings.Trim(request.FormValue("redirect"), "")
		if redirect == "" {
			redirect = "/home"
		}
		channel := make(chan models.SignInResponse) // Create a channel to receive the sign-in response
		defer close(channel)
		go services.SignInEmailPassword(email, password, channel)
		signInResponse := <-channel // Wait for the sign-in response

		if email == "" || password == "" || signInResponse.IDToken == "ERROR" {
			sse := datastar.NewSSE(responseWriter, request)
			sse.MergeSignals([]byte("{errorMessage:'Error. Invalid Credentials'}"))
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
	})
	router.Get("/home", func(responseWriter http.ResponseWriter, request *http.Request) {
		popularsChannel := make(chan models.PopularsFromDb)
		defer close(popularsChannel)
		go services.GetPopulars(request.Context(), popularsChannel)
		populars := <-popularsChannel

		component := components.Home(populars.Data)
		component.Render(request.Context(), responseWriter)
	})
	router.Get("/data/{ticker}", func(responseWriter http.ResponseWriter, request *http.Request) {
		ticker := strings.Trim(chi.URLParam(request, "ticker"), "")
		if ticker == "" {
			http.Error(responseWriter, "Ticker not provided", http.StatusBadRequest)
			return
		}

		sse := datastar.NewSSE(responseWriter, request)
		cachedDataChannel := make(chan []models.CacheData)
		defer close(cachedDataChannel)
		go services.GetCachedData(ticker, cachedDataChannel)
		chartData := <-cachedDataChannel

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
				sse.MergeFragmentTempl(components.CardError(ticker))
				return
			} else {
				setCacheChannel := make(chan string)
				defer close(setCacheChannel)
				go services.SetCachedData(ticker, chartData, setCacheChannel)
				<-setCacheChannel
			}
		}

		eChartDataChannel := make(chan models.EChartData)
		defer close(eChartDataChannel)
		go getEchartData(chartData, eChartDataChannel)
		eChartData := <-eChartDataChannel

		str := `LoadChart("chart_` + ticker + `",[` + eChartData.AxisData + `],[` + eChartData.ChartData + `])`
		// fmt.Println("Sending data to client:", str)
		sse.ExecuteScript(str, datastar.WithExecuteScriptAutoRemove(true))
	})

	router.Get("/assets/*", func(responseWriter http.ResponseWriter, request *http.Request) {
		http.StripPrefix("/assets/", http.FileServer(http.Dir("assets/"))).ServeHTTP(responseWriter, request)
	})
	http.ListenAndServe(":3000", router)
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
