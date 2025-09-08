package main

import (
	"bytes"
	"context"
	"datastar-stock/components"
	"datastar-stock/models"
	"datastar-stock/services"
	"fmt"
	"os"
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
		emailData[idx] = models.EmailPopularsPriceData{
			Ticker: ticker,
		}
		if len(popularsDict[ticker]) >= 1 {
			latestDayData := popularsDict[ticker][len(popularsDict[ticker])-1]
			emailData[idx].Date = latestDayData.Date
			if val, err := strconv.ParseFloat(latestDayData.Close, 64); err == nil {
				emailData[idx].Price = val
			}
			if len(popularsDict[ticker]) >= 2 {
				prevDayData := popularsDict[ticker][len(popularsDict[ticker])-2]
				if val, err := strconv.ParseFloat(prevDayData.Close, 64); err == nil {
					emailData[idx].PrevPrice = val
				}
				if emailData[idx].Price >= emailData[idx].PrevPrice {
					emailData[idx].IsIncrease = true
				} else {
					emailData[idx].IsIncrease = false
				}
			} else {
				emailData[idx].IsIncrease = true
			}

		}
	}
	emailStrBuffer := new(bytes.Buffer)
	components.EmailTemplate(emailData).Render(context.Background(), emailStrBuffer)
	emailChannel := make(chan string)
	defer close(emailChannel)
	go sendEmail(models.SendEmail{To: os.Getenv("EMAIL_TO"), From: os.Getenv("EMAIL_FROM"), HtmlBody: emailStrBuffer.String(), Subject: os.Getenv("EMAIL_SUBJECT"), APIKey: os.Getenv("RESEND_API_KEY")}, emailChannel)
	fmt.Println(<-emailChannel)

	//For debugging uncommen the below
	//fmt.Println(emailStrBuffer.String())
}

func getData(ticker string, channel chan<- []models.CacheData) {
	defer close(channel)
	waitForSetCache := false
	setCacheChannel := make(chan string)
	defer close(setCacheChannel)

	chartData := services.CallStockApisInPriority(ticker)

	if len(chartData) > 0 {
		go services.SetCacheTickerData(models.CacheKey{Ticker: ticker, Date: time.Now().Format("2006-01-02")}, chartData, setCacheChannel)
		waitForSetCache = true
	} else {
		cachedDataTodayChannel := make(chan []models.CacheData)
		defer close(cachedDataTodayChannel)
		go services.GetCachedTickerData(models.CacheKey{Ticker: ticker, Date: time.Now().Format("2006-01-02")}, cachedDataTodayChannel)
		chartData := <-cachedDataTodayChannel
		if len(chartData) == 0 {
			cachedDataPrevDayChannel := make(chan []models.CacheData)
			defer close(cachedDataPrevDayChannel)
			go services.GetCachedTickerData(models.CacheKey{Ticker: ticker, Date: time.Now().AddDate(0, 0, -1).Format("2006-01-02")}, cachedDataPrevDayChannel)
			chartData = <-cachedDataPrevDayChannel
		}

	}
	if waitForSetCache {
		<-setCacheChannel
	}
	channel <- chartData
}
func sendEmail(model models.SendEmail, channel chan<- string) {
	client := resend.NewClient(model.APIKey)
	params := &resend.SendEmailRequest{
		From:    model.From,
		To:      []string{model.To},
		Html:    model.HtmlBody,
		Subject: model.Subject,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		fmt.Printf("Error sending Email %v\n", err.Error())
		channel <- "Error sending Email"
		return
	}
	channel <- "Email successfully sent"
}
