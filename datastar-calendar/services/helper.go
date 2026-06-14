package services

type contextKey string

const TokenKey contextKey = "token"

var AllowedFrequencies = []string{"Only once", "Daily", "Weekly", "Every two weeks", "Monthly", "Every two months", "Quarterly", "Half yearly", "Yearly"}

func GenerateUserSessionKey(accessToken string, sId string) string {
	return accessToken + "-" + sId
}
