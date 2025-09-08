package models

type ClientSignals struct {
	DefaultVal string `json:"defaultSelection"`
}

type Loader struct {
	DataShowSignal string
	Class          string
	ContainerClass string
}

type CityLatLng struct {
	City string
	Lat  string
	Lng  string
}
