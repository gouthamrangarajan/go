package models

type OpenRouterResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type OpenRouterRequest struct {
	Model    string                     `json:"model"`
	Messages []OpenRouterRequestMessage `json:"messages"`
}

type OpenRouterRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QuizResponse struct {
	VideoId       string
	Summary       string `json:"summary"`
	TalkingPoints []struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	} `json:"talkingPoints"`
	Questions []struct {
		Id             string   `json:"id"`
		Type           string   `json:"type"`
		Difficulty     string   `json:"difficulty"`
		Question       string   `json:"question"`
		ShortAnswer    string   `json:"shortAnswer"`
		SpeakingAnswer string   `json:"speakingAnswer"`
		KeyTerms       []string `json:"keyTerms"`
		SourceExcerpt  string   `json:"sourceExcerpt"`
		StartSeconds   string   `json:"startSeconds"`
	} `json:"questions"`
}
