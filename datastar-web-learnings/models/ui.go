package models

type UISignals struct {
	IdToken        string   `json:"idToken"`
	Tags           []string `json:"tags"`
	VideoId        string   `json:"videoId"`
	Title          string   `json:"title"`
	Subtitle       string   `json:"subtitle"`
	Rank           int      `json:"rank"`
	Transcript     string   `json:"transcript"`
	VideoToDelete  string   `json:"videoToDelete"`
	SearchTxt      string   `json:"searchTxt"`
	Sid            string   `json:"sid"`
	Offset         int      `json:"offset"`
	QuizVideoId    string   `json:"quizVideoId"`
	QuizVideoTitle string   `json:"quizVideoTitle"`
	QuizIndex      int      `json:"quizIndex"`
}
