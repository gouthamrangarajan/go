package models

import "time"

type NoteData struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content_editorjs"`
	UpdatedAt time.Time `json:"updated_at"`
	UserId    string    `json:"user_id"`
	Order     int       `json:"order"`
}

type OTPVerificationResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}
