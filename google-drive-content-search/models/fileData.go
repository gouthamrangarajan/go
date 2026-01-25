package models

type FileData struct {
	Name                  string
	Id                    string
	Data                  []byte
	MimeType              string
	ExtractedText         string
	OCRText               string
	ExtractedTextMarkdown string
}
