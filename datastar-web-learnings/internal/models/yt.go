package models

type YoutubeVideoSearchResponse struct {
	Items []struct {
		Id      string `json:"id"`
		Snippet struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"snippet"`
	} `json:"items"`
}
