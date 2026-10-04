package openrouter

type Request struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
}

type RequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DecisionRequest struct {
	Model     string                             `json:"model"`
	State     map[string]string                  `json:"state"`
	Questions map[string]DecisionRequestQuestion `json:"questions"`
}

type DecisionRequestQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria"` // NoulCriteria | map[string]string | []string
}

type NoulCriteria struct {
	True  string `json:"true"`
	False string `json:"false"`
}
