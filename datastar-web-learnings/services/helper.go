package services

import (
	"datastar-web-learnings/models"
	"fmt"
	"strings"
)

const VECTOR_DATA_TEMPLATE = `Title: %v
Subtitle: %v
Tags: %v
Description: %v`

func ConstructTextToVectorize(data models.VideoResponse, description string) models.VideoResponse {
	data.TextToVectorize = fmt.Sprintf(VECTOR_DATA_TEMPLATE, data.Title, data.Subtitle, strings.Join(data.Tags, ", "), description)
	data.DescriptionFromYTAPI = description
	return data
}
