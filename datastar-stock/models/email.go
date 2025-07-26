package models

type EmailPopularsPriceData struct {
	Ticker     string
	Date       string
	Price      float64
	PrevPrice  float64
	IsIncrease bool
}
