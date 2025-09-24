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

type ReorderNoteInfo struct {
	Id       string `json:"id"`
	NewIndex int    `json:"newIndex"`
	OldIndex int    `json:"oldIndex"`
}

type ReorderNote struct {
	Info ReorderNoteInfo `json:"orderInfo"`
}
