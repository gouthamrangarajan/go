package main

import (
	"context"
	"datastar-stock/components"
	"datastar-stock/components/shared"
	"datastar-stock/models"
	"datastar-stock/services"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

func tickerDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.TrimSpace(chi.URLParam(request, "ticker"))
	if ticker == "" {
		http.Error(responseWriter, "Ticker not provided", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(responseWriter, request)
	ticketDataHandlerWithSSE(ticker, sse)
}
func ticketDataHandlerWithSSE(ticker string, sse *datastar.ServerSentEventGenerator) {
	cachedDataTodayChannel := make(chan []models.CacheData)
	defer close(cachedDataTodayChannel)

	waitForSetCache := false
	setCacheChannel := make(chan string)
	defer close(setCacheChannel)

	go services.GetCachedTickerData(ticker, time.Now().Format("2006-01-02"), cachedDataTodayChannel)
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
			go services.GetCachedTickerData(ticker, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), cachedDataPrevDayChannel)
			chartData = <-cachedDataPrevDayChannel
			if len(chartData) == 0 { //still no data
				if apiData.ErrorMessage != "" && strings.Contains(strings.ToLower(apiData.ErrorMessage), "invalid api call") {
					sse.PatchElementTempl(shared.CardTickerError(ticker, "Error! Invalid Ticker"))
				} else {
					sse.PatchElementTempl(shared.CardTickerError(ticker, "Error! Try again later"))
				}
				return
			}
		} else {
			go services.SetCacheTickerData(ticker, time.Now().Format("2006-01-02"), chartData, setCacheChannel)
			waitForSetCache = true
		}
	}

	eChartDataChannel := make(chan models.EChartData)
	defer close(eChartDataChannel)
	go getEchartData(chartData, eChartDataChannel)
	eChartData := <-eChartDataChannel

	str := `LoadChart("chart_` + shared.ReplaceSpecialCharsInTicker(ticker) + `",[` + eChartData.AxisData + `],[` + eChartData.ChartData + `])`
	sse.ExecuteScript(str, datastar.WithExecuteScriptAutoRemove(true))
	populars := getPopulars(sse.Context())
	if slices.Contains(populars, ticker) {
		sse.PatchElementTempl(shared.TickerIsInPopularUI(ticker))
	} else {
		sse.PatchElementTempl(shared.TickerIsNotInPopularUI(ticker))
	}
	if waitForSetCache {
		<-setCacheChannel
	}
}
func multipleTickerDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	tickers := strings.TrimSpace(chi.URLParam(request, "tickers"))
	if tickers == "" {
		http.Error(responseWriter, "Tickers not provided", http.StatusBadRequest)
		return
	}
	tickerList := strings.Split(tickers, "||")
	if len(tickerList) == 0 {
		http.Error(responseWriter, "Ticker not provided", http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(responseWriter, request)

	for _, ticker := range tickerList {
		ticketDataHandlerWithSSE(ticker, sse)
	}

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
			eChartData.ChartData += `[` + value.Open + `,` + value.Close + `,` + value.Low + `,` + value.High + `]`
		} else {
			eChartData.AxisData += `'` + value.Date + `'` + ","
			eChartData.ChartData += `[` + value.Open + `,` + value.Close + `,` + value.Low + `,` + value.High + `]` + ","
		}
	}
	channel <- eChartData
}
func getPopulars(ctx context.Context) []string {
	popularsCacheChannel := make(chan []string)
	defer close(popularsCacheChannel)
	go services.GetCachedPopularsData(popularsCacheChannel)
	popularsCache := <-popularsCacheChannel
	if len(popularsCache) > 0 {
		return popularsCache
	}
	popularsChannel := make(chan models.PopularsFromDb)
	defer close(popularsChannel)
	go services.GetPopulars(ctx, popularsChannel)
	populars := <-popularsChannel

	popularsCacheSaveChannel := make(chan string)
	defer close(popularsCacheSaveChannel)

	go services.SetCachePopularsData(populars.Data, popularsCacheSaveChannel)
	<-popularsCacheSaveChannel
	return populars.Data
}
func popularsPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	populars := getPopulars(request.Context())
	component := components.PopularsError()
	if len(populars) > 0 {
		cardList := make([]models.TickerCard, len(populars))
		for idx, ticker := range populars {
			cardList[idx] = models.TickerCard{
				Ticker: ticker,
				Name:   "",
			}
		}
		component = components.Populars(cardList)
	}
	component.Render(request.Context(), responseWriter)
}
func popularsPriorityIncrementDecrementHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.TrimSpace(request.FormValue("ticker"))
	incrementDecrementIndicator := strings.TrimSpace(request.FormValue("incrementDecrementIndicator"))

	if ticker == "" || (incrementDecrementIndicator != "increase" && incrementDecrementIndicator != "decrease") {
		http.Error(responseWriter, "Bad request", http.StatusBadRequest)
		return
	}
	populars := getPopulars(request.Context())
	idx := slices.Index(populars, ticker)
	if idx == -1 {
		http.Error(responseWriter, "Bad request", http.StatusBadRequest)
		return
	}
	if incrementDecrementIndicator == "increase" {
		if idx == 0 {
			http.Error(responseWriter, "Bad request", http.StatusBadRequest)
			return
		}
		populars[idx], populars[idx-1] = populars[idx-1], populars[idx]
	} else {
		if idx == len(populars)-1 {
			http.Error(responseWriter, "Bad request", http.StatusBadRequest)
			return
		}
		populars[idx], populars[idx+1] = populars[idx+1], populars[idx]
	}
	sse := datastar.NewSSE(responseWriter, request)
	cardList := make([]models.TickerCard, len(populars))
	for idx, ticker := range populars {
		cardList[idx] = models.TickerCard{
			Ticker: ticker,
			Name:   "",
		}
	}
	sse.PatchElementTempl(components.PopularsContainers(cardList, false), datastar.WithUseViewTransitions(true))
	saveChannel := make(chan bool)
	go services.SetPopulars(request.Context(), populars, saveChannel)

	popularsCacheSaveChannel := make(chan string)
	defer close(popularsCacheSaveChannel)
	go services.SetCachePopularsData(populars, popularsCacheSaveChannel)

	time.Sleep(300 * time.Millisecond) // wait for cards to be available
	for _, tickerInPopulars := range populars {
		ticketDataHandlerWithSSE(tickerInPopulars, sse)
	}
	<-saveChannel
	<-popularsCacheSaveChannel
}
func popularsConfigureUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	channel := make(chan []models.RecentFromDb)
	go services.GetRecent(request.Context(), channel)

	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.PopularsConfigure(), datastar.WithModeAppend(), datastar.WithSelector("body"), datastar.WithUseViewTransitions(true))
	time.Sleep(300 * time.Millisecond) // wait for the modal to be available
	sse.ExecuteScript("confineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))

	populars := getPopulars(request.Context()) // prefetch populars data
	popularsToSend := make([]models.TickerCard, len(populars))
	for idx, ticker := range populars {
		popularsToSend[idx] = models.TickerCard{
			Ticker: strings.TrimSpace(ticker),
			Name:   "",
		}
	}
	sse.PatchElementTempl(components.PopularsConfigureCards(popularsToSend, "populars"), datastar.WithSelectorID("popularsConfigure"), datastar.WithModeAppend())
	sse.PatchElementTempl(components.AvailablePopularsCount(len(populars)))
	recents := <-channel
	recentsToSend := []models.TickerCard{}
	for _, item := range recents {
		if len(recentsToSend) > 24 {
			break
		}
		if slices.Contains(populars, strings.TrimSpace(item.Ticker)) {
			continue
		}
		recentsToSend = append(recentsToSend, models.TickerCard{Ticker: strings.TrimSpace(item.Ticker), Name: strings.TrimSpace(item.Name)})
	}
	sse.PatchElementTempl(components.PopularsConfigureCards(recentsToSend, "recent"), datastar.WithSelectorID("recentPopularsConfigure"), datastar.WithModeAppend())
}
func closeConfigurePopularsHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
	sse.RemoveElement("#overlay", datastar.WithUseViewTransitions(true))
}
func recentPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	recentsChannel := make(chan []models.RecentFromDb)
	defer close(recentsChannel)
	go services.GetRecent(request.Context(), recentsChannel)
	recents := <-recentsChannel

	recentsToSend := []models.TickerCard{}
	for idx, item := range recents {
		if idx > 4 {
			break
		}
		recentsToSend = append(recentsToSend, models.TickerCard{Ticker: strings.TrimSpace(item.Ticker), Name: strings.TrimSpace(item.Name)})
	}
	component := components.RecentError()
	if len(recentsToSend) > 0 {
		component = components.Recent(recentsToSend)
	}
	component.Render(request.Context(), responseWriter)
}

func recentDataHandlerWithCount(responseWriter http.ResponseWriter, request *http.Request) {
	newCountStr := strings.TrimSpace(request.FormValue("newCount"))
	currentCountStr := strings.TrimSpace(request.FormValue("currentCount"))

	newCount, err := strconv.Atoi(newCountStr)
	if err != nil || newCount < 1 {
		http.Error(responseWriter, "Invalid request", http.StatusBadRequest)
		return
	}
	currentCount, err := strconv.Atoi(currentCountStr)
	if err != nil || currentCount < 0 {
		http.Error(responseWriter, "Invalid request", http.StatusBadRequest)
		return
	}

	offset := currentCount
	numberOfItemsToSend := newCount - currentCount
	sse := datastar.NewSSE(responseWriter, request)
	recentsChannel := make(chan []models.RecentFromDb)
	defer close(recentsChannel)
	go services.GetRecent(request.Context(), recentsChannel)
	recents := <-recentsChannel
	if len(recents) == 0 {
		// sse.MergeFragmentTempl(components.RecentError(), datastar.WithUseViewTransitions(true))
		// sse.MergeFragmentTempl(components.CurrentCountInp(0))
		sse.PatchSignals([]byte("{loading:false}"))
		return
	}
	if numberOfItemsToSend > 0 {
		recentsToSend := []models.TickerCard{}
		recentsToSendIndex := 0
		for idx, item := range recents {
			if idx >= newCount {
				break
			} else if idx+1 <= offset {
				continue
			}

			recentsToSend = append(recentsToSend, models.TickerCard{Ticker: strings.TrimSpace(item.Ticker), Name: strings.TrimSpace(item.Name)})
			recentsToSendIndex++
		}
		sse.PatchElementTempl(shared.Cards(recentsToSend, "recent"), datastar.WithSelectorID("recents"), datastar.WithModeAppend(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.CurrentCountInp(newCount))
		time.Sleep(300 * time.Millisecond) // wait for cards to be available
		sse.ExecuteScript(`document.getElementById('card_`+shared.ReplaceSpecialCharsInTicker(recentsToSend[0].Ticker)+`')?.scrollIntoView({behavior: 'smooth', block: 'nearest'});`, datastar.WithExecuteScriptAutoRemove(true))
		for _, tickerInRecent := range recentsToSend {
			ticketDataHandlerWithSSE(tickerInRecent.Ticker, sse)
		}
	} else {
		for idx := range currentCount - newCount {
			if idx+newCount >= len(recents) {
				break
			}
			tickerToRemove := strings.TrimSpace(recents[idx+newCount].Ticker)
			// sse.ExecuteScript(`DisposeChart("chart_`+tickerToRemove+`")`, datastar.WithExecuteScriptAutoRemove(true))
			sse.RemoveElement(`#card_`+shared.ReplaceSpecialCharsInTicker(tickerToRemove), datastar.WithUseViewTransitions(true))
		}
		sse.PatchElementTempl(components.CurrentCountInp(newCount))
	}
}

func addRecentUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.AddRecent(), datastar.WithModeAppend(), datastar.WithSelector("body"), datastar.WithUseViewTransitions(true))
	time.Sleep(300 * time.Millisecond) // wait for the modal to be available
	sse.ExecuteScript("confineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
}

func closeAddRecentHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
	sse.RemoveElement("#companies_tbody")
	sse.RemoveElement("#overlay", datastar.WithUseViewTransitions(true))
}

func addRecentTickerHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.TrimSpace(chi.URLParam(request, "ticker"))
	company := strings.TrimSpace(chi.URLParam(request, "company"))
	if ticker == "" {
		http.Error(responseWriter, "Bad request", http.StatusBadRequest)
		return
	}
	if company != "" {
		replacer1 := strings.NewReplacer("||||", "%")
		replacer2 := strings.NewReplacer("|||", "$")
		replacer3 := strings.NewReplacer("||", "/")
		replacer4 := strings.NewReplacer("%20", " ")
		company = replacer1.Replace(company)
		company = replacer2.Replace(company)
		company = replacer3.Replace(company)
		company = replacer4.Replace(company)
	}

	currentCountStr := strings.TrimSpace(request.FormValue("currentCount"))
	currentCount, _ := strconv.Atoi(currentCountStr)

	recentFromDbChannel := make(chan []models.RecentFromDb)
	defer close(recentFromDbChannel)
	go services.GetRecent(request.Context(), recentFromDbChannel)
	recents := <-recentFromDbChannel

	recentWithCurrentCount := make([]models.RecentFromDb, currentCount)
	recentAlreadyContainsTicker := false

	for idx, item := range recents {
		if idx >= currentCount {
			break
		}
		recentWithCurrentCount[idx] = item
		if item.Ticker == ticker {
			recentAlreadyContainsTicker = true
		}
	}

	addRecentToDbChannel := make(chan bool)
	defer close(addRecentToDbChannel)
	go services.AddRecent(request.Context(), ticker, company, addRecentToDbChannel)

	sse := datastar.NewSSE(responseWriter, request)
	if recentAlreadyContainsTicker {
		sse.RemoveElement("#card_" + shared.ReplaceSpecialCharsInTicker(ticker))
	} else {
		sse.RemoveElement("#card_" + shared.ReplaceSpecialCharsInTicker(recentWithCurrentCount[len(recentWithCurrentCount)-1].Ticker))
	}
	sse.PatchElementTempl(shared.Card(models.TickerCard{Ticker: ticker, Name: company}), datastar.WithSelector(".card:first-child"), datastar.WithModeBefore())
	<-addRecentToDbChannel
}
