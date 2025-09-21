package models

type UICookie struct {
	Name  string
	Value string
}

type UINote struct {
	Id      string `json:"noteId"`
	Content string `json:"noteContent"`
	Title   string `json:"noteTitle"`
}
