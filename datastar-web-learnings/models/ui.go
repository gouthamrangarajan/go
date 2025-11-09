package models

type UISignals struct {
	IdToken    string   `json:"idToken"`
	Tags       []string `json:"tags"`
	VideoId    string   `json:"videoId"`
	Title      string   `json:"title"`
	Subtitle   string   `json:"subtitle"`
	Rank       int      `json:"rank"`
	Transcript string   `json:"transcript"`
}
