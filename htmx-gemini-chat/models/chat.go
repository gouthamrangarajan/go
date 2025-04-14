package models

type ChatSession struct {
	Id    int64
	Title string
}
type ChatConversation struct {
	Id        int64
	SessionId int64
	Message   string
	Sender    string
}
