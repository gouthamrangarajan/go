package models

type DnD struct {
	Id   string `json:"id"`
	Date string `json:"date"`
}

type DeleteEvent struct {
	Id          string
	AccessToken string
}

type MonthYearDayWeekString struct {
	Month string
	Year  string
	Week  string
	Day   string
}
