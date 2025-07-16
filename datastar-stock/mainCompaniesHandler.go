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
func sseZeroOffsetCompaniesList(sse *datastar.ServerSentEventGenerator, companies []models.CompanyFromDb) {
	limitStr := os.Getenv("COMPANIES_LIMIT")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100 // default limit
	}
	endIndex := 0 + limit
	if endIndex > len(companies) {
		endIndex = len(companies)
	}
	sse.PatchElementTempl(shared.CompaniesTr(companies[0:endIndex], "companies"), datastar.WithSelectorID("companies_tbody"), datastar.WithModeAppend())
	sse.PatchElementTempl(shared.LoadMore("@get('/companies/all/" + strconv.Itoa(endIndex) + "')"))
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
	if searchTerm != "" && len(searchTerm) < 3 {
		sse.PatchElementTempl(shared.CompaniesTbodyHint(page), datastar.WithUseViewTransitions(useViewTransition))
		return
	}

	companiesSaveCacheChannel := make(chan string)
	defer close(companiesSaveCacheChannel)

	companies, saveCacheCalled := getAllCompanies(request.Context(), companiesSaveCacheChannel)
	if page == "companies" && searchTerm == "" {
		sseZeroOffsetCompaniesList(sse, companies)
	} else {
		companies = filterCompaniesBySearchTerm(companies, searchTerm)
		if len(companies) == 0 {
			sse.PatchElementTempl(shared.CompaniesTbodyEmpty(page), datastar.WithUseViewTransitions(useViewTransition))
		} else {
			sse.PatchElementTempl(shared.CompaniesTbody(companies, page), datastar.WithUseViewTransitions(useViewTransition))
		}
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

func companiesPageHandler(responseWriter http.ResponseWriter, request *http.Request) {
	component := components.Companies()
	component.Render(request.Context(), responseWriter)
}

func companiesAllDataHandler(responseWriter http.ResponseWriter, request *http.Request) {
	offsetStr := strings.TrimSpace(chi.URLParam(request, "offset"))
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limitStr := os.Getenv("COMPANIES_LIMIT")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100 // default limit
	}

	companiesSaveCacheChannel := make(chan string)
	defer close(companiesSaveCacheChannel)

	companies, saveCacheCalled := getAllCompanies(request.Context(), companiesSaveCacheChannel)

	sse := datastar.NewSSE(responseWriter, request)
	if offset >= len(companies) {
		sse.PatchElementTempl(shared.LoadMoreNoAction())
		return
	} else if offset == 0 {
		sse.PatchElementTempl(components.CompaniesCount(len(companies)))
	}

	endIndex := offset + limit
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

		sseZeroOffsetCompaniesList(sse, companies)
	}
	<-saveCacheChannel
}
