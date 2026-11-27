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
	managePageFilter models.ManagerPageFilters) models.ManageEventsPageData {
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

	return pageData
}
