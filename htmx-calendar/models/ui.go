package models

import "time"

type GoToDay struct {
	GoToMonth int
	GoToYear  int
	GoToDay   int
	CurrMonth int
	CurrYear  int
	CurrDay   int
}

type TableTdTemplateData struct {
	Row     int
	Col     int
	From    string
	DaysLen int
	DataLen int
}

type TableTdData struct {
	Data                time.Time
	CurrentMonthAndYear time.Time
}
