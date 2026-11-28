package services

import (
	"datastar-calendar/internal/models"
	"time"
)

type contextKey string

const TokenKey contextKey = "token"

var AllowedFrequencies = []string{"Only once", "Daily", "Weekly", "Every two weeks", "Monthly", "Every two months", "Quarterly", "Half yearly", "Yearly"}

func GenerateUserSessionKey(accessToken string, sId string) string {
	return accessToken + "-" + sId
}

func GenerateAllDatesStringFromStartToEnd(start time.Time, end time.Time) []string {
	ret := []string{}
	loopDt := start
	ret = append(ret, loopDt.Format("2006-01-02"))
	for {
		loopDt = loopDt.AddDate(0, 0, 1)
		if end.Sub(loopDt) < 0 {
			break
		}
		ret = append(ret, loopDt.Format("2006-01-02"))
	}
	return ret
}

func ConstructManageEventsPageData(dbEventsData []models.EventData,
	managePageFilter models.ManagerPageFilters, pageNumber int) (models.ManageEventsPageData, int) {
	var pageData models.ManageEventsPageData
	for _, item := range dbEventsData {
		if item.IsActive {
			pageData.ActiveDataCount++
		} else {
			pageData.InactiveDataCount++
		}
		switch {
		case managePageFilter == models.ACTIVE_STATUS && item.IsActive:
			pageData.EventsData = append(pageData.EventsData, item)
		case managePageFilter == models.INACTIVE_STATUS && !item.IsActive:
			pageData.EventsData = append(pageData.EventsData, item)
		case managePageFilter == models.ALL_STATUS:
			pageData.EventsData = append(pageData.EventsData, item)
		}
	}
	totalRecords := len(pageData.EventsData)

	// fmt.Printf("%v\n", totalRecords)
	if len(pageData.EventsData) > 0 {
		startNum := (pageNumber - 1) * models.NO_OF_RECORDS_PER_PAGE
		if startNum >= len(pageData.EventsData) {
			startNum = len(pageData.EventsData) - 1
		}
		endNum := startNum + models.NO_OF_RECORDS_PER_PAGE
		if endNum > len(pageData.EventsData) {
			endNum = len(pageData.EventsData)
		}
		// fmt.Printf("%v,%v\n", startNum, endNum)
		pageData.EventsData = pageData.EventsData[startNum:endNum]
	}
	return pageData, totalRecords
}
