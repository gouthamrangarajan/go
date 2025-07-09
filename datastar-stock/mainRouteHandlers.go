package main

import (
	"datastar-stock/components"
	"datastar-stock/components/shared"
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
				sse.MergeFragmentTempl(shared.CardTickerError(ticker))
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

	str := `LoadChart("chart_` + shared.ReplaceSpecialCharsInTicker(ticker) + `",[` + eChartData.AxisData + `],[` + eChartData.ChartData + `])`
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
	component := components.PopularsError()
	if len(populars.Data) > 0 {
		component = components.Populars(populars.Data)
	}
	component.Render(request.Context(), responseWriter)

}

func recentDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	recentsChannel := make(chan []models.RecentFromDb)
	defer close(recentsChannel)
	go services.GetRecent(request.Context(), recentsChannel)
	recents := <-recentsChannel

	recentsToSend := []string{}
	for idx, item := range recents {
		if idx > 4 {
			break
		}
		recentsToSend = append(recentsToSend, strings.Trim(item.Ticker, " "))
	}
	component := components.RecentError()
	if len(recentsToSend) > 0 {
		component = components.Recent(recentsToSend)

	}
	component.Render(request.Context(), responseWriter)
}

func recentDataHandlerWithCount(responseWriter http.ResponseWriter, request *http.Request) {
	newCountStr := strings.Trim(request.FormValue("newCount"), "")
	currentCountStr := strings.Trim(request.FormValue("currentCount"), "")

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
		sse.MergeSignals([]byte("{loading:false}"))
		return
	}
	if numberOfItemsToSend > 0 {
		recentsToSend := []string{}
		recentsToSendIndex := 0
		for idx, item := range recents {
			if idx >= newCount {
				break
			} else if idx+1 <= offset {
				continue
			}

			recentsToSend = append(recentsToSend, strings.Trim(item.Ticker, " "))
			recentsToSendIndex++
		}
		sse.MergeFragmentTempl(shared.Cards(recentsToSend), datastar.WithSelectorID("recents"), datastar.WithMergeAppend(), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript(`document.getElementById('card_`+shared.ReplaceSpecialCharsInTicker(recentsToSend[0])+`').scrollIntoView({behavior: 'smooth', block: 'center'});`, datastar.WithExecuteScriptAutoRemove(true))
	} else {
		for idx := range currentCount - newCount {
			if idx+newCount >= len(recents) {
				break
			}
			tickerToRemove := strings.Trim(recents[idx+newCount].Ticker, " ")
			// sse.ExecuteScript(`DisposeChart("chart_`+tickerToRemove+`")`, datastar.WithExecuteScriptAutoRemove(true))
			sse.RemoveFragments(`#card_`+shared.ReplaceSpecialCharsInTicker(tickerToRemove), datastar.WithRemoveUseViewTransitions(true))
		}
	}
	sse.MergeFragmentTempl(components.CurrentCountInp(newCount))
	sse.MergeSignals([]byte("{loading:false}"))
}

func addRecentUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.MergeFragmentTempl(components.AddRecent(), datastar.WithMergeAppend(), datastar.WithSelector("body"), datastar.WithUseViewTransitions(true))
}

func searchCompaniesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	searchTerm := strings.Trim(request.FormValue("search"), "")
	companies := []models.CompanyFromDb{}
	if searchTerm != "" {
		companiesChannel := make(chan []models.CompanyFromDb)
		defer close(companiesChannel)
		go services.GetAllCompanies(request.Context(), companiesChannel)
		companies = <-companiesChannel
		companies = filterCompaniesBySearchTerm(companies, searchTerm)
	}
	sse := datastar.NewSSE(responseWriter, request)
	if len(companies) == 0 {
		sse.MergeFragmentTempl(shared.CompaniesTbodyEmpty())
	} else {
		sse.MergeFragmentTempl(shared.CompaniesTbody(companies))
	}
	sse.MergeSignals([]byte("{searching:false}"))
}

func filterCompaniesBySearchTerm(companies []models.CompanyFromDb, searchTerm string) []models.CompanyFromDb {
	filteredCompanies := []models.CompanyFromDb{}
	searchTerm = strings.ToLower(searchTerm)

	for _, company := range companies {
		if strings.Contains(strings.ToLower(company.Name), searchTerm) || strings.Contains(strings.ToLower(company.Ticker), searchTerm) {
			filteredCompanies = append(filteredCompanies, company)
		}
	}

	sort.Slice(filteredCompanies, func(a, b int) bool {
		return filteredCompanies[a].Name < filteredCompanies[b].Name
	})

	return filteredCompanies
}

func closeAddRecentHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.RemoveFragments("#overlay", datastar.WithRemoveUseViewTransitions(true))
}

func addRecentTickerHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.Trim(chi.URLParam(request, "ticker"), "")
	if ticker == "" {
		http.Error(responseWriter, "Bad request", http.StatusBadRequest)
		return
	}
	name := strings.Trim(chi.URLParam(request, "name"), "")

	currentCountStr := strings.Trim(request.FormValue("currentCount"), "")
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
	go services.AddRecent(request.Context(), ticker, name, addRecentToDbChannel)

	sse := datastar.NewSSE(responseWriter, request)
	if recentAlreadyContainsTicker {
		sse.RemoveFragments("#card_" + shared.ReplaceSpecialCharsInTicker(ticker))
	} else {
		sse.RemoveFragments("#card_" + shared.ReplaceSpecialCharsInTicker(recentWithCurrentCount[len(recentWithCurrentCount)-1].Ticker))
	}
	sse.MergeFragmentTempl(shared.CardToPrepend(ticker), datastar.WithSelector(".card:first-child"), datastar.WithMergeBefore())

	sse.MergeSignals([]byte("{adding_" + shared.ReplaceSpecialCharsInTicker(ticker) + ":false}"))

	<-addRecentToDbChannel
}
