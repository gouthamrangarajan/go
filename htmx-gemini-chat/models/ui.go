package models

type Section struct {
	ChatSessionId   int
	WebSearch       bool
	ImageGeneration bool
	IsOob           bool
	HelperTextShow  bool
	MenuSrchTxt     string
}

type PromptInput struct {
	SessionId      int
	Prompt         string
	FileBase64     string
	AllowWebSearch bool
}

type ChatTemplate struct {
	SessionId      int
	ConversationId int
}
