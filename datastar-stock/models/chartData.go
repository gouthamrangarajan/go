package models

type CacheData struct {
	Date   string `json:"date"`
	Open   string `json:"open"`
	Close  string `json:"close"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Volume string `json:"volume"`
}

type EChartData struct {
	AxisData  string
	ChartData string
}
