package models

type ClientSignals struct {
	Id            string   `json:"id"`
	ServiceFilter []string `json:"serviceFilter"`
	SrchTxt       string   `json:"srchTxt"`
	Email         string   `json:"email"`
	Message       string   `json:"message"`
}

type SearchSuggestion struct {
	Suggestion string `json:"suggestion"`
	Tag        string `json:"tag"`
}

type ProjectDetail struct {
	Id                 int
	Title              string
	ImgSrc             string
	Description        string
	Url                string
	Service            string
	Tags               string
	ImgBadgeLightMode  bool
	CodeUrl            string
	AISuggestionTag    string
	AISuggestionReason string
}
