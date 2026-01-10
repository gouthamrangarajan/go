package services

import (
	"database/sql"
	"datastar-openrouter/models"
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
		fmt.Printf("Failed to execute query in GetAllChatSessionsForJob: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title); err != nil {
			fmt.Printf("Error scanning row in GetAllChatSessionsForJob:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- data
}
func GetAiModels(channel chan<- []models.AIModel) {
	var data []models.AIModel = []models.AIModel{}
	db, err := createDb()
	if err != nil {
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT model_id,model_display_name FROM models WHERE is_active=1 ORDER BY sort_order")
	if err != nil {
		fmt.Printf("Failed to execute query in GetAllModels: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.AIModel

		if err := rows.Scan(&item.ModelId, &item.DisplayName); err != nil {
			fmt.Printf("Error scanning row in GetAllModels:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetAllModels:%v\n", err.Error())
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
	rows, err := db.Query("SELECT session_id,title,allow_web_search,img_generation FROM chat_sessions WHERE user_id = ? ORDER BY session_id", userId)
	if err != nil {
		fmt.Printf("Failed to execute query in GetChatSessions: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title, &item.AllowWebSearch, &item.ImageGeneration); err != nil {
			fmt.Printf("Error scanning row in GetChatSessions:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetChatSessions:%v\n", err.Error())
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
		fmt.Printf("Error marshalling search title query vector in SearchChatSessions: %v\n", err.Error())
		channel <- data
		return
	}
	vectorStr := string(vectorStrBytes)
	rows, err := db.Query("SELECT session_id,title,allow_web_search FROM chat_sessions WHERE user_id = ? AND vector_distance_cos(title_vector, vector32(?)) < 0.8 ORDER BY vector_distance_cos(title_vector, vector32(?))", userId, vectorStr, vectorStr)
	if err != nil {
		fmt.Printf("Failed to execute query in SearchChatSessions: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatSession

		if err := rows.Scan(&item.Id, &item.Title, &item.AllowWebSearch); err != nil {
			fmt.Printf("Error scanning row in SearchChatSessions:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in SearchChatSessions:%v\n", err.Error())
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
	result, err := db.Exec("INSERT INTO chat_sessions (user_id,title,allow_web_search,img_generation,created_at) VALUES (?, ?,?,?,?)", userId, data.Title, data.AllowWebSearch, data.ImageGeneration, time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute query in InsertChatSession: %v\n", err.Error())
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Printf("Error getting last inserted id in InsertChatSession: %v\n", errInsertId.Error())
		channel <- 0
		return
	}
	channel <- int(newId)
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
		fmt.Printf("Failed to execute query in UpdateChatSessionTitle: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating title in UpdateChatSessionTitle: %v\n", errUpdate.Error())
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
		fmt.Printf("Failed to execute query in UpdateChatSessionTitleVector: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating title vector in UpdateChatSessionTitleVector : %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	// fmt.Printf("Updated title vector for session id: %d\n", sessionId)
	channel <- int(rowsAffected)
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
		fmt.Printf("Failed to execute query in UpdateChatSessionAllowWebSearch: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating allow_web_search in UpdateChatSessionAllowWebSearch: %v\n", errUpdate.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func UpdateChatSessionImageGeneration(userId string, sessionId int, imageGeneration bool, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE chat_sessions SET img_generation = ? WHERE session_id = ? AND  user_id = ?", imageGeneration, sessionId, userId)
	if err != nil {
		fmt.Printf("Failed to execute query in UpdateChatSessionImageGeneration: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error updating allow_web_search in UpdateChatSessionImageGeneration: %v\n", errUpdate.Error())
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
		fmt.Printf("Failed to execute query in DeleteChatSession: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Printf("Error deleting Chat Session in DeleteChatSession : %v\n", errUpdate.Error())
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
	rows, err := db.Query("SELECT DISTINCT conversation_id,chat_conversations.session_id,content,role,model_id,file_name,file_data FROM chat_conversations INNER JOIN chat_sessions ON chat_sessions.session_id=chat_conversations.session_id WHERE chat_sessions.session_id = ? AND user_id=? ORDER BY timestamp", sessionId, userId)
	if err != nil {
		fmt.Printf("Failed to execute query in GetChatConversations: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.ChatConversation

		if err := rows.Scan(&item.Id, &item.SessionId, &item.Content, &item.Role, &item.ModelId, &item.FileName, &item.FileData); err != nil {
			fmt.Printf("Error scanning row in GetChatConversations:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetChatConversations:%v\n", err.Error())
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
	result, err := db.Exec("INSERT INTO chat_conversations (session_id,content,role,model_id,file_name,file_data,timestamp) VALUES (?, ?,?,?,?,?,?)", data.SessionId, data.Content, data.Role, data.ModelId, data.FileName, data.FileData, time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute query in InsertChatConversation: %v\n", err.Error())
		channel <- 0
		return
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Printf("Error getting last inserted id in InsertChatConversation: %v\n", errInsertId.Error())
		channel <- 0
		return
	}
	channel <- int(newId)
}
func UpateMessageChatConversation(data models.UpdateChatConversation, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE chat_conversations SET content= ?,model_id=?,file_data=? WHERE  conversation_id=?", data.Content, data.ModelId, data.FileData, data.Id)
	if err != nil {
		fmt.Printf("Failed to execute query in UpateMessageChatConversation: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error updating chat conversation in UpateMessageChatConversation: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func DeleteMessageChatConversation(conversationId int, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM chat_conversations WHERE  conversation_id=?", conversationId)
	if err != nil {
		fmt.Printf("Failed to execute query in DeleteMessageChatConversation: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Error deleting chat conversation in DeleteMessageChatConversation: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func DeleteMessageChatConversationForRetry(data models.DeleteChatConversationsAfterAId, channel chan<- []int) {
	db, err := createDb()
	returnIds := []int{}
	if err != nil {
		channel <- returnIds
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT conversation_id FROM chat_conversations WHERE session_id=? AND conversation_id>? ORDER BY conversation_id", data.SessionId, data.ConversationIdAfterWhichDelete)
	if err != nil {
		fmt.Printf("Failed to execute query in DeleteMessageChatConversationForRetry: %v\n", err.Error())
		channel <- returnIds
		return
	}
	defer rows.Close()
	for rows.Next() {
		var conversationId int
		if err := rows.Scan(&conversationId); err != nil {
			fmt.Printf("Error scanning row in DeleteMessageChatConversationForRetry:%v\n", err.Error())
		} else {
			returnIds = append(returnIds, conversationId)
		}
	}
	result, err := db.Exec("DELETE FROM chat_conversations  WHERE session_id=? AND conversation_id>? ", data.SessionId, data.ConversationIdAfterWhichDelete)
	if err != nil {
		fmt.Printf("Failed to execute query in DeleteMessageChatConversationForRetry: %v\n", err.Error())
		channel <- []int{}
		return
	}
	_, err = result.RowsAffected()
	if err != nil {
		fmt.Printf("Error deleting chat conversations in DeleteMessageChatConversationForRetry: %v\n", err.Error())
		channel <- []int{}
		return
	}
	channel <- returnIds
}
