package models

type WorldCities struct {
	Id      int
	City    string
	Lat     string
	Lng     string
	Country string
	State   string
}

type TourismSpots struct {
	Id       int
	Name     string
	Lat      string
	Lng      string
	NearCity string
	NearLat  string
	NearLng  string
	UnixTime int64
}
