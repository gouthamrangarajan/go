package main

import (
	"bytes"
	"context"
	"datastar-stock/components"
	"datastar-stock/models"
	"datastar-stock/services"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/resend/resend-go/v2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	popularsChannel := make(chan []string)
	defer close(popularsChannel)
	go services.GetCachedPopularsData(popularsChannel)
	populars := <-popularsChannel
	// fmt.Println(populars)
	// populars = []string{"NFLX", "META"}
	channels := make([]chan []models.CacheData, len(populars))
	for idx := range channels {
		channels[idx] = make(chan []models.CacheData)
	}
	for idx, ticker := range populars {
		go getData(ticker, channels[idx])
	}

	popularsDict := make(map[string][]models.CacheData, len(populars))
	for idx, channel := range channels {
		for data := range channel {
			popularsDict[populars[idx]] = data
		}
	}

	emailData := make([]models.EmailPopularsPriceData, len(populars))

	for idx, ticker := range populars {
		latestDayData := popularsDict[ticker][len(popularsDict[ticker])-1]
		prevDayData := popularsDict[ticker][len(popularsDict[ticker])-2]
		emailData[idx] = models.EmailPopularsPriceData{
			Ticker: ticker,
			Date:   latestDayData.Date,
		}
		if val, err := strconv.ParseFloat(latestDayData.Close, 64); err == nil {
			emailData[idx].Price = val
		}
		if val, err := strconv.ParseFloat(prevDayData.Close, 64); err == nil {
			emailData[idx].PrevPrice = val
		}
		if emailData[idx].Price >= emailData[idx].PrevPrice {
			emailData[idx].IsIncrease = true
		} else {
			emailData[idx].IsIncrease = false
		}
	}
	emailStrBuffer := new(bytes.Buffer)
	components.EmailTemplate(emailData).Render(context.Background(), emailStrBuffer)
	emailChannel := make(chan string)
	defer close(emailChannel)
	go sendEmail(os.Getenv("EMAIL_TO"), os.Getenv("EMAIL_FROM"), emailStrBuffer.String(), os.Getenv("EMAIL_SUBJECT"), os.Getenv("RESEND_API_KEY"), emailChannel)
	fmt.Println(<-emailChannel)
}

func getData(ticker string, channel chan<- []models.CacheData) {
	defer close(channel)
	waitForSetCache := false
	setCacheChannel := make(chan string)
	defer close(setCacheChannel)

	chartData := make([]models.CacheData, 0)
	alphavantageChannel := make(chan models.AlphavantageResponse)
	defer close(alphavantageChannel)
	go services.CallAlphavantageAPI(ticker, alphavantageChannel)
	apiData := <-alphavantageChannel
	dates := make([]string, 0, len(apiData.TimeSeriesDaily))
	for date := range apiData.TimeSeriesDaily {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	for _, date := range dates {
		dailyData := apiData.TimeSeriesDaily[date]
		chartData = append(chartData, models.CacheData{
			Date:   date,
			Close:  dailyData.Close,
			Open:   dailyData.Open,
			High:   dailyData.High,
			Low:    dailyData.Low,
			Volume: dailyData.Volume,
		})
	}
	if len(chartData) > 0 {
		go services.SetCacheTickerData(ticker, time.Now().Format("2006-01-02"), chartData, setCacheChannel)
		waitForSetCache = true
	} else {
		cachedDataTodayChannel := make(chan []models.CacheData)
		defer close(cachedDataTodayChannel)
		go services.GetCachedTickerData(ticker, time.Now().Format("2006-01-02"), cachedDataTodayChannel)
		chartData := <-cachedDataTodayChannel
		if len(chartData) == 0 {
			cachedDataPrevDayChannel := make(chan []models.CacheData)
			defer close(cachedDataPrevDayChannel)
			go services.GetCachedTickerData(ticker, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), cachedDataPrevDayChannel)
			chartData = <-cachedDataPrevDayChannel
		}

	}
	if waitForSetCache {
		<-setCacheChannel
	}
	channel <- chartData
}
func sendEmail(to string, from string, htmlBody string, subject string, apiKey string, channel chan<- string) {
	client := resend.NewClient(apiKey)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Html:    htmlBody,
		Subject: subject,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Printf("Error sending Email %v\n", err.Error())
		channel <- "Error sending Email"
		return
	}
	channel <- "Email successfully sent"
}
