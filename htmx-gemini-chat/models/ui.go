package models

type Section struct {
	ChatSessionId   int
	WebSearch       bool
	ImageGeneration bool
	IsOob           bool
	HelperTextShow  bool
}

type PromptInput struct {
	SessionId      int
	Prompt         string
	FileBase64     string
	AllowWebSearch bool
}
