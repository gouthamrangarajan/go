package models

type ChatSession struct {
	Id              int
	Title           string
	TitleVector     []float32
	AllowWebSearch  bool
	ImageGeneration bool
}
type ChatConversation struct {
	Id        int
	SessionId int
	Content   string
	Role      string
	ModelId   string
	FileName  string
	FileData  string
}

type UpdateChatConversation struct {
	Id       int
	Content  string
	ModelId  string
	FileData string
	FileName string
}

type AIModel struct {
	ModelId     string
	DisplayName string
	IsDefault   bool
}

type DeleteChatConversationsAfterAId struct {
	SessionId                      int
	ConversationIdAfterWhichDelete int
	UserId                         string
}
