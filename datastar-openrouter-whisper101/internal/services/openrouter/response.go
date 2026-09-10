package openrouter

type TranscribeResponse struct {
	Text  string `json:"text"`
	Usage struct {
		Seconds     float64 `json:"seconds"`
		TotalTokens int     `json:"total_tokens"`
		Cost        float64 `json:"cost"`
	} `json:"usage"`
}
