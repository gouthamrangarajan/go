package models

type CompanyTable struct {
	Page  string
	Class string
}

type Spinner struct {
	DataShowSignal string
	Class          string
}

type UICard struct {
	Index          int
	Page           string
	TotalNoOfItems int
}

type TickerSequence struct {
	Index            int
	Ticker           string
	TotalNoOfTickers int
}

type TickerError struct {
	Ticker       string
	ErrorMessage string
}

type EChartData struct {
	AxisData  string
	ChartData string
}
