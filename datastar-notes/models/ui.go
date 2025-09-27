package models

type UICookie struct {
	Name  string
	Value string
}
type OTPForm struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	Message string `json:"message"`
	IsError bool   `json:"isError"`
}
type UINote struct {
	Id      string `json:"noteId"`
	Content string `json:"noteContent"`
	Title   string `json:"noteTitle"`
}

type ReorderNoteInfo struct {
	Id       string `json:"id"`
	NewIndex int    `json:"newIndex"`
	OldIndex int    `json:"oldIndex"`
}

type ReorderNote struct {
	Info ReorderNoteInfo `json:"orderInfo"`
}
