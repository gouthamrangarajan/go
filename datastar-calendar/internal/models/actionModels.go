package models

type DeleteEvent struct {
	Id          string
	AccessToken string
}

type UpdateIsActiveEvent struct {
	Id          string
	AccessToken string
	IsActive    bool
}

type MonthYearDayWeekString struct {
	Month string
	Year  string
	Week  string
	Day   string
}
