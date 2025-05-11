package services

import (
	"database/sql"
	"fmt"
	"htmx-gemini-chat/models"
	"os"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func createDb() *sql.DB {
	dbUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	url := fmt.Sprintf("%v?authToken=%v", dbUrl, authToken)

	db, err := sql.Open("libsql", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", url, err)
		os.Exit(1)
	}
	return db
}

func InsertUser(userId string, channel chan<- int) {
	db := createDb()
	defer db.Close()
	result, err := db.Exec("INSERT INTO users (user_id,created_at) VALUES (?,?)", userId, time.Now().Unix())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute user insert query: %v\n", err)
		channel <- 0
	}
	rowsAffected, errInsert := result.RowsAffected()
	if errInsert != nil {
		fmt.Fprintf(os.Stderr, "Error getting rows affected for user insert: %v\n", errInsert)
		channel <- 0
	}
	channel <- int(rowsAffected)
}

func GetChatSessions(userId string, channel chan<- []models.ChatSession) {
	db := createDb()
	defer db.Close()
	var data []models.ChatSession = []models.ChatSession{}
	rows, err := db.Query("SELECT session_id,title FROM chat_sessions WHERE user_id = ?", userId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title); err != nil {
			fmt.Println("Error scanning row:", err)
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
	}
	channel <- data
}

func InsertChatSession(userId string, title string, channel chan<- int) {
	db := createDb()
	defer db.Close()
	result, err := db.Exec("INSERT INTO chat_sessions (user_id,title, created_at) VALUES (?, ?,?)", userId, title, time.Now().Unix())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Fprintf(os.Stderr, "Error getting last inserted id: %v\n", errInsertId)
		channel <- 0
		return
	}
	channel <- int(newId)
}

func UpdateChatSessionTitle(userId string, sessionId int, title string, channel chan<- int) {
	db := createDb()
	defer db.Close()
	result, err := db.Exec("UPDATE chat_sessions SET title = ? WHERE session_id = ? AND  user_id = ?", title, sessionId, userId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Fprintf(os.Stderr, "Error updating title : %v\n", errUpdate)
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func DeleteChatSession(userId string, sessionId int, channel chan<- int) {
	db := createDb()
	defer db.Close()
	result, err := db.Exec("DELETE FROM chat_sessions WHERE session_id = ? AND  user_id = ?", sessionId, userId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Fprintf(os.Stderr, "Error deleting Chat Session : %v\n", errUpdate)
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func GetChatConversations(userId string, sessionId int, channel chan<- []models.ChatConversation) {
	db := createDb()
	defer db.Close()
	var data []models.ChatConversation = []models.ChatConversation{}
	rows, err := db.Query("SELECT DISTINCT conversation_id,chat_conversations.session_id,message,sender,img_data FROM chat_conversations INNER JOIN chat_sessions ON chat_sessions.session_id=chat_conversations.session_id WHERE chat_sessions.session_id = ? AND user_id=? ORDER BY timestamp", sessionId, userId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatConversation

		if err := rows.Scan(&item.Id, &item.SessionId, &item.Message, &item.Sender, &item.ImgData); err != nil {
			fmt.Println("Error scanning row:", err)
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
	}
	channel <- data
}

func InsertChatConversation(sessionId int, message string, imgData string, sender string, channel chan<- int) {
	db := createDb()
	defer db.Close()
	result, err := db.Exec("INSERT INTO chat_conversations (session_id,message,sender,img_data, timestamp) VALUES (?, ?,?,?,?)", sessionId, message, sender, imgData, time.Now().Unix())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Fprintf(os.Stderr, "Error getting last inserted id: %v\n", errInsertId)
		channel <- 0
		return
	}
	channel <- int(newId)
}
