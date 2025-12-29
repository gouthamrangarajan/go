package models

type ChatSession struct {
	Id             int
	Title          string
	TitleVector    []float32
	AllowWebSearch bool
}
type ChatConversation struct {
	Id        int
	SessionId int
	Content   string
	Role      string
	ModelId   string
}

type UpdateChatConversation struct {
	Id      int
	Content string
	ModelId string
}
