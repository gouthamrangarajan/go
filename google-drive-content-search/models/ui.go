package models

type ClientSignals struct {
	Query string `json:"query"`
	Token string `json:"token"`
}

type SearchResult struct {
	Id                  string
	FileName            string
	MatchingContent     string
	MatchPercent        string
	FileContentMarkdown string
}
