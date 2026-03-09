package models

type JsonData struct {
	AllDemoItems []struct {
		Order             int      `json:"order"`
		Title             string   `json:"title"`
		ImgSrc            string   `json:"imgSrc"`
		Description       string   `json:"description"`
		Url               string   `json:"url"`
		Tags              []string `json:"tags"`
		Service           string   `json:"service"`
		Display           bool     `json:"display"`
		ImgBadgeLightMode bool     `json:"imgBadgeLightMode"`
	} `json:"all"`
}
