package services

import (
	"database/sql"
	"datastar-claude-chat/models"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func createDb() (*sql.DB, error) {
	dbUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	url := fmt.Sprintf("%v?authToken=%v", dbUrl, authToken)

	db, err := sql.Open("libsql", url)
	if err != nil {
		fmt.Printf("Failed to open db %v: %v", url, err.Error())
	}
	return db, err
}
func CheckUserExistsInTable(userId string, channel chan<- bool) {
	db, err := createDb()
	if err != nil {
		channel <- false
		return
	}
	defer db.Close()
	rows, err := db.Query("select 1 from users where user_Id=? LIMIT 1", userId)
	if err != nil {
		fmt.Printf("Failed to execute check user query: %v\n", err.Error())
		channel <- false
		return
	}
	defer rows.Close()
	id := 0
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	if id != 0 {
		channel <- true
	} else {
		channel <- false
	}

}
func InsertUser(userId string, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("INSERT INTO users (user_id,created_at) VALUES (?,?)", userId, time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute user insert query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errInsert := result.RowsAffected()
	if errInsert != nil {
		fmt.Printf("Error getting rows affected for user insert: %v\n", errInsert.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func GetAllChatSessionsForJob(channel chan<- []models.ChatSession) {
	var data []models.ChatSession = []models.ChatSession{}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT session_id,title FROM chat_sessions")
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}

func GetChatSessions(userId string, channel chan<- []models.ChatSession) {
	var data []models.ChatSession = []models.ChatSession{}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT session_id,title,allow_web_search FROM chat_sessions WHERE user_id = ? ORDER BY session_id", userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title, &item.AllowWebSearch); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}
func SearchChatSessions(userId string, searchVector []float32, channel chan<- []models.ChatSession) {
	var data []models.ChatSession = []models.ChatSession{}
	if len(searchVector) == 0 {
		fmt.Printf("Ignoring search title due to empty vector\n")
		channel <- data
		return
	}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	vectorStrBytes, err := json.Marshal(searchVector)
	if err != nil {
		fmt.Printf("Error marshalling search title query vector: %v\n", err.Error())
		channel <- data
		return
	}
	vectorStr := string(vectorStrBytes)
	rows, err := db.Query("SELECT session_id,title,allow_web_search FROM chat_sessions WHERE user_id = ? AND vector_distance_cos(title_vector, vector32(?)) < 0.5 ORDER BY vector_distance_cos(title_vector, vector32(?))", userId, vectorStr, vectorStr)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title, &item.AllowWebSearch); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}
func InsertChatSession(userId string, data models.ChatSession, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("INSERT INTO chat_sessions (user_id,title,allow_web_search, created_at) VALUES (?, ?,?,?)", userId, data.Title, data.AllowWebSearch, time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Printf("Error getting last inserted id: %v\n", errInsertId.Error())
		channel <- 0
		return
	}
	channel <- int(newId)
}
func UpdateChatSessionAllowWebSearch(userId string, sessionId int, webSearch bool, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE chat_sessions SET allow_web_search = ? WHERE session_id = ? AND  user_id = ?", webSearch, sessionId, userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating allow_web_search : %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func UpdateChatSessionTitle(userId string, data models.ChatSession, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE chat_sessions SET title = ? WHERE session_id = ? AND  user_id = ?", data.Title, data.Id, userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating title : %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func UpdateChatSessionTitleVector(sessionId int, titleVector []float32, channel chan<- int) {
	if len(titleVector) == 0 {
		fmt.Printf("Ignoring title vector update for session id: %d due to empty vector\n", sessionId)
		channel <- 0
		return
	}
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	vectorStrBytes, err := json.Marshal(titleVector)
	if err != nil {
		fmt.Printf("Error marshalling title vector: %v\n", err.Error())
		channel <- 0
		return
	}
	result, err := db.Exec("UPDATE chat_sessions SET title_vector = vector32(?) WHERE session_id = ?", string(vectorStrBytes), sessionId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating title vector : %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func DeleteChatSession(userId string, sessionId int, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM chat_sessions WHERE session_id = ? AND  user_id = ?", sessionId, userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error deleting Chat Session : %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func GetChatConversations(userId string, sessionId int, channel chan<- []models.ChatConversation) {
	var data []models.ChatConversation = []models.ChatConversation{}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT DISTINCT conversation_id,chat_conversations.session_id,message,sender,img_data,pdf_data,file_id,file_name FROM chat_conversations INNER JOIN chat_sessions ON chat_sessions.session_id=chat_conversations.session_id WHERE chat_sessions.session_id = ? AND user_id=? ORDER BY timestamp", sessionId, userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatConversation

		if err := rows.Scan(&item.Id, &item.SessionId, &item.Message, &item.Sender, &item.ImgData, &item.PdfData, &item.FileId, &item.FileName); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}
func GetChatConversation(userId string, conversationId int, session models.ChatSession, channel chan<- models.ChatConversation) {
	var data models.ChatConversation = models.ChatConversation{}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT DISTINCT conversation_id,chat_conversations.session_id,message,sender,img_data,pdf_data,file_id,file_name FROM chat_conversations INNER JOIN chat_sessions ON chat_sessions.session_id=chat_conversations.session_id WHERE conversation_id=? AND chat_sessions.session_id = ? AND user_id=? ORDER BY timestamp", conversationId, session.Id, userId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatConversation

		if err := rows.Scan(&item.Id, &item.SessionId, &item.Message, &item.Sender, &item.ImgData, &item.PdfData, &item.FileId, &item.FileName); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			data = item
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}

func InsertChatConversation(data models.ChatConversation, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("INSERT INTO chat_conversations (session_id,message,sender,img_data,pdf_data,file_id,file_name, timestamp) VALUES (?, ?,?,?,?,?,?,?)", data.SessionId, data.Message, data.Sender, data.ImgData, data.PdfData, data.FileId, data.FileName, time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Printf("Error getting last inserted id: %v\n", errInsertId.Error())
		channel <- 0
		return
	}
	channel <- int(newId)
}
func UpateMessageChatConversation(conversationId int, message string, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE chat_conversations SET message= ? WHERE  conversation_id=?", message, conversationId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error updating chat conversation: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func DeleteClaudeMessageChatConversation(conversationId int, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM chat_conversations WHERE  conversation_id=?", conversationId)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error deleting chat conversation: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
