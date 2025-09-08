package models

type CacheData struct {
	Date   string `json:"date"`
	Open   string `json:"open"`
	Close  string `json:"close"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Volume string `json:"volume"`
}

type CacheKey struct {
	Ticker string
	Date   string
}

type Tokens struct {
	IdToken      string
	RefreshToken string
}
