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
	Message   string
	Sender    string
	ImgData   string
	FileId    string
	FileName  string
}
