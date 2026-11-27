package services

import "time"

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
