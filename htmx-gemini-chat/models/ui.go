package models

type Section struct {
	ChatSessionId   int
	WebSearch       bool
	ImageGeneration bool
	IsOob           bool
	HelperTextShow  bool
}
