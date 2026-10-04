package openrouter

type Response struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
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

type AnswerEvaluationResponse struct {
	IsTechnicallyCorrect bool     `json:"is_technically_correct"`
	AccuracyScore        float32  `json:"accuracy_score"`
	FluencyScore         float32  `json:"fluency_score"`
	UsedKeywords         []string `json:"used_keywords"`
	MissingKeywords      []string `json:"missing_keywords"`
	FeedbackTip          string   `json:"feedback_tip"`
	ImprovedSpokenAnswer string   `json:"improved_spoken_answer"`
}

type DecisionsResponse struct {
	Model    string            `json:"model"`
	Answers  map[string]Answer `json:"answers"`
	Usage    Usage             `json:"usage"`
	ID       string            `json:"id"`
	Provider string            `json:"provider"`
}

type Answer struct {
	Type string `json:"type"`

	// Noul is populated for "noul" answers.
	Noul *float64 `json:"noul,omitempty"`

	// Choice and Probabilities are populated for "choice" answers.
	Choice        *string            `json:"choice,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`

	// Confidence may be present for choice answers.
	Confidence *float64 `json:"confidence,omitempty"`
}

type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}
