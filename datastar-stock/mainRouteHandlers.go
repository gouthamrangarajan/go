package main

import (
	"context"
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
	"github.com/starfederation/datastar/sdk/go/datastar"
)

func landingPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	component := components.Landing()
	component.Render(request.Context(), responseWriter)
}
func loginHandler(responseWriter http.ResponseWriter, request *http.Request) {
	email := strings.TrimSpace(request.FormValue("email"))
	password := request.FormValue("password")
	redirect := strings.TrimSpace(request.FormValue("redirect"))
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
		sse.PatchElementTempl(shared.FormSubmitEmptyResult(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(shared.FormSubmitResult("Error! Please provide valid Email & Password", true), datastar.WithUseViewTransitions(true))
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
		sse := datastar.NewSSE(responseWriter, request)
		sse.PatchElementTempl(shared.FormSubmitEmptyResult(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(shared.FormSubmitResult("Successfully logged in.", false), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.LoginInSubmitBtn(true), datastar.WithUseViewTransitions(true))
		sse.ExecuteScript("window.location.href = window.location.href", datastar.WithExecuteScriptAutoRemove(true))
	}
}
func tickerDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.TrimSpace(chi.URLParam(request, "ticker"))
	if ticker == "" {
		http.Error(responseWriter, "Ticker not provided", http.StatusBadRequest)
		return
	}

	sse := datastar.NewSSE(responseWriter, request)
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
			go services.SetCachedTickerData(ticker, time.Now().Format("2006-01-02"), chartData, setCacheChannel)
			waitForSetCache = true
		}
	}

	eChartDataChannel := make(chan models.EChartData)
	defer close(eChartDataChannel)
	go getEchartData(chartData, eChartDataChannel)
	eChartData := <-eChartDataChannel

	str := `LoadChart("chart_` + shared.ReplaceSpecialCharsInTicker(ticker) + `",[` + eChartData.AxisData + `],[` + eChartData.ChartData + `])`
	sse.ExecuteScript(str, datastar.WithExecuteScriptAutoRemove(true))
	if waitForSetCache {
		<-setCacheChannel
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

func popularsPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
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

func recentPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	recentsChannel := make(chan []models.RecentFromDb)
	defer close(recentsChannel)
	go services.GetRecent(request.Context(), recentsChannel)
	recents := <-recentsChannel

	recentsToSend := []string{}
	for idx, item := range recents {
		if idx > 4 {
			break
		}
		recentsToSend = append(recentsToSend, strings.TrimSpace(item.Ticker))
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
		recentsToSend := []string{}
		recentsToSendIndex := 0
		for idx, item := range recents {
			if idx >= newCount {
				break
			} else if idx+1 <= offset {
				continue
			}

			recentsToSend = append(recentsToSend, strings.TrimSpace(item.Ticker))
			recentsToSendIndex++
		}
		sse.PatchElementTempl(shared.Cards(recentsToSend), datastar.WithSelectorID("recents"), datastar.WithModeAppend(), datastar.WithUseViewTransitions(true))
		sse.PatchElementTempl(components.CurrentCountInp(newCount))
		time.Sleep(300 * time.Millisecond) //wait for the cards to be rendered
		sse.ExecuteScript(`document.getElementById('card_`+shared.ReplaceSpecialCharsInTicker(recentsToSend[0])+`')?.scrollIntoView({behavior: 'smooth', block: 'nearest'});`, datastar.WithExecuteScriptAutoRemove(true))
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
func getAllCompanies(ctx context.Context, companiesSaveCacheChannel chan string) ([]models.CompanyFromDb, bool) {
	companies := []models.CompanyFromDb{}
	companiesCacheChannel := make(chan []models.CompanyFromDb)
	defer close(companiesCacheChannel)

	saveCacheCalled := false

	go services.GetCachedCompaniesData(companiesCacheChannel)
	companies = <-companiesCacheChannel
	if len(companies) == 0 {
		companiesChannel := make(chan []models.CompanyFromDb)
		defer close(companiesChannel)
		go services.GetAllCompanies(ctx, companiesChannel)
		companies = <-companiesChannel

		go services.SetCachedCompaniesData(companies, companiesSaveCacheChannel)
		saveCacheCalled = true
	}
	return companies, saveCacheCalled
}
func searchCompaniesHandler(responseWriter http.ResponseWriter, request *http.Request) {
	searchTerm := strings.TrimSpace(request.FormValue("search"))
	page := strings.TrimSpace(request.FormValue("page"))

	useViewTransition := false

	// if page == "home" {
	// 	useViewTransition = false
	// }

	sse := datastar.NewSSE(responseWriter, request)
	if page == "companies" {
		sse.PatchElementTempl(shared.LoadMoreNoAction())
	}
	if searchTerm == "" || len(searchTerm) < 3 {
		sse.PatchElementTempl(shared.CompaniesTbodyHint(page), datastar.WithUseViewTransitions(useViewTransition))
		if page == "companies" && searchTerm == "" {
			sse.PatchElementTempl(shared.LoadMore("@get('/companies/all/0')"))
		}
		return
	}

	companiesSaveCacheChannel := make(chan string)
	defer close(companiesSaveCacheChannel)

	companies, saveCacheCalled := getAllCompanies(request.Context(), companiesSaveCacheChannel)
	companies = filterCompaniesBySearchTerm(companies, searchTerm)
	if len(companies) == 0 {
		sse.PatchElementTempl(shared.CompaniesTbodyEmpty(page), datastar.WithUseViewTransitions(useViewTransition))
	} else {
		sse.PatchElementTempl(shared.CompaniesTbody(companies, page), datastar.WithUseViewTransitions(useViewTransition))
	}
	if saveCacheCalled {
		<-companiesSaveCacheChannel
	}

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
	sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
	sse.RemoveElement("#companies_tbody")
	sse.RemoveElement("#overlay", datastar.WithUseViewTransitions(true))
}

func addRecentTickerHandler(responseWriter http.ResponseWriter, request *http.Request) {
	ticker := strings.TrimSpace(chi.URLParam(request, "ticker"))
	if ticker == "" {
		http.Error(responseWriter, "Bad request", http.StatusBadRequest)
		return
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
	go services.AddRecent(request.Context(), ticker, addRecentToDbChannel)

	sse := datastar.NewSSE(responseWriter, request)
	if recentAlreadyContainsTicker {
		sse.RemoveElement("#card_" + shared.ReplaceSpecialCharsInTicker(ticker))
	} else {
		sse.RemoveElement("#card_" + shared.ReplaceSpecialCharsInTicker(recentWithCurrentCount[len(recentWithCurrentCount)-1].Ticker))
	}
	sse.PatchElementTempl(shared.Card(ticker), datastar.WithSelector(".card:first-child"), datastar.WithModeBefore())
	<-addRecentToDbChannel
}
func companiesPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	component := components.Companies()
	component.Render(request.Context(), responseWriter)
}
func companiesCountHandler(responseWriter http.ResponseWriter, request *http.Request) {
	companiesSaveCacheChannel := make(chan string)
	defer close(companiesSaveCacheChannel)
	companies, saveCacheCalled := getAllCompanies(request.Context(), companiesSaveCacheChannel)

	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.CompaniesCount(len(companies)), datastar.WithUseViewTransitions(true))

	if saveCacheCalled {
		<-companiesSaveCacheChannel
	}
}
func companiesAllDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	offsetStr := strings.TrimSpace(chi.URLParam(request, "offset"))
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	companiesSaveCacheChannel := make(chan string)
	defer close(companiesSaveCacheChannel)

	companies, saveCacheCalled := getAllCompanies(request.Context(), companiesSaveCacheChannel)

	sse := datastar.NewSSE(responseWriter, request)
	if offset >= len(companies) {
		sse.PatchElementTempl(shared.LoadMoreNoAction())
		return
	}

	endIndex := offset + 100
	if endIndex > len(companies) {
		endIndex = len(companies)
	}

	sse.PatchElementTempl(shared.CompaniesTr(companies[offset:endIndex], "companies"), datastar.WithSelectorID("companies_tbody"), datastar.WithModeAppend())
	sse.PatchElementTempl(shared.LoadMore("@get('/companies/all/" + strconv.Itoa(endIndex) + "')"))
	if saveCacheCalled {
		<-companiesSaveCacheChannel
	}

}
func addCompanyUIHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.PatchElementTempl(components.AddCompany(), datastar.WithModeAppend(), datastar.WithSelector("body"), datastar.WithUseViewTransitions(true))
	time.Sleep(300 * time.Millisecond) // wait for the modal to be available
	sse.ExecuteScript("confineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
}
func closeAddCompanyHandler(responseWriter http.ResponseWriter, request *http.Request) {
	sse := datastar.NewSSE(responseWriter, request)
	sse.ExecuteScript("removeConfineFocusToModal()", datastar.WithExecuteScriptAutoRemove(true))
	sse.RemoveElement("#overlay", datastar.WithUseViewTransitions(true))
}

func addCompanyHandler(responseWriter http.ResponseWriter, request *http.Request) {
	name := strings.TrimSpace(request.FormValue("name"))
	ticker := strings.ToUpper(strings.TrimSpace(request.FormValue("ticker")))
	sse := datastar.NewSSE(responseWriter, request)

	sse.PatchElementTempl(shared.FormSubmitEmptyResult())
	if name == "" || ticker == "" || len(ticker) < 3 {
		sse.PatchElementTempl(shared.FormSubmitResult("Error! Please provide valid Ticker & Name", true))
		return
	}

	saveCacheDuringGetChannel := make(chan string)
	defer close(saveCacheDuringGetChannel)

	companies, saveCacheCalledDuringGet := getAllCompanies(request.Context(), saveCacheDuringGetChannel)
	if saveCacheCalledDuringGet {
		<-saveCacheDuringGetChannel
	}
	maxId := 1
	for _, company := range companies {
		if company.Ticker == ticker {
			sse.PatchElementTempl(shared.FormSubmitResult("Error! Ticker already exists", true))
			return
		} else if company.Id > maxId {
			maxId = company.Id
		}
	}
	company := models.CompanyFromDb{Id: maxId + 1, Ticker: ticker, Name: name}

	companies = append(companies, company)

	saveDbChannel := make(chan bool)
	defer close(saveDbChannel)
	go services.SetAllCompanies(request.Context(), companies, saveDbChannel)

	saveCacheChannel := make(chan string)
	defer close(saveCacheChannel)
	go services.SetCachedCompaniesData(companies, saveCacheChannel)

	saveDbSuccessful := <-saveDbChannel

	if !saveDbSuccessful {
		sse.PatchElementTempl(shared.FormSubmitResult("Error! Please try again later", true))
	} else {
		sse.PatchElementTempl(shared.FormSubmitResult(`Ticker `+ticker+` successfully added`, false))
		sse.PatchElementTempl(components.CompaniesCount(len(companies)))
		sse.ExecuteScript("document.getElementById('addCompanyForm')?.reset();", datastar.WithExecuteScriptAutoRemove(true))
		sse.PatchElementTempl(shared.CompaniesTbodyHint("companies"))
		sse.ExecuteScript("document.getElementById('companySearchForm')?.reset();", datastar.WithExecuteScriptAutoRemove(true))
	}
	<-saveCacheChannel
}
