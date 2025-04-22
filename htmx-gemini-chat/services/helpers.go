package services

import "htmx-gemini-chat/models"

func getChatSessionsViaChannel(userId string) []models.ChatSession {
	sessionChannel := make(chan []models.ChatSession)
	defer close(sessionChannel)
	go GetChatSessions(userId, sessionChannel)
	sessions := <-sessionChannel
	return sessions
}
func insertChatSessionViaChannel(userId string, title string) int {
	var sessionId int = 0
	insertSessionChannel := make(chan int)
	defer close(insertSessionChannel)
	go InsertChatSession(userId, title, insertSessionChannel)
	sessionId = <-insertSessionChannel
	return sessionId
}
