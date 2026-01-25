package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"google-drive-content-search/models"
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
func DeleteAllData(channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM document_chunks")
	if err != nil {
		fmt.Printf("Failed to execute query in DeleteAllData: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected in DeleteAllData: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func DeleteData(fileName string, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("DELETE FROM document_chunks WHERE file_name = ?", fileName)
	if err != nil {
		fmt.Printf("Failed to execute query in DeleteData: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected in DeleteData: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func InsertData(data models.DocumentChunk, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		channel <- 0
		return
	}
	defer db.Close()
	vectorStrBytes, err := json.Marshal(data.ChunkEmbedding)
	if err != nil {
		fmt.Printf("Error marshalling vector during insert: %v\n", err.Error())
		channel <- 0
		return
	}
	result, err := db.Exec("INSERT INTO document_chunks (file_id,file_name,mime_type,full_content_markdown,chunk_content,chunk_embeddings,updated_at) VALUES (?, ?,?,?,?,vector32(?),?)",
		data.FileId, data.FileName, data.MimeType, data.FullContentMarkdown, data.ChunkContent, string(vectorStrBytes), time.Now().Unix())
	if err != nil {
		fmt.Printf("Failed to execute query in InsertData: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected in InsertData: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func SearchData(embedding []float32, channel chan<- []models.DocumentChunk) {
	returnData := []models.DocumentChunk{}
	db, err := createDb()
	if err != nil {
		channel <- returnData
		return
	}
	defer db.Close()
	vectorStrBytes, err := json.Marshal(embedding)
	if err != nil {
		fmt.Printf("Error marshalling vector during search: %v\n", err.Error())
		channel <- returnData
		return
	}
	vectorStr := string(vectorStrBytes)
	rows, err := db.Query("SELECT id,file_id,file_name,mime_type,chunk_content,full_content_markdown,vector_distance_cos(chunk_embeddings, vector32(?)) FROM document_chunks WHERE vector_distance_cos(chunk_embeddings, vector32(?)) < 0.8 ORDER BY vector_distance_cos(chunk_embeddings, vector32(?))", vectorStr, vectorStr, vectorStr)
	if err != nil {
		fmt.Printf("Failed to execute query in SeachData: %v\n", err.Error())
		channel <- returnData
		return
	}
	defer rows.Close()
	for rows.Next() {
		var docChunk models.DocumentChunk
		err := rows.Scan(&docChunk.Id, &docChunk.FileId, &docChunk.FileName, &docChunk.MimeType, &docChunk.ChunkContent, &docChunk.FullContentMarkdown, &docChunk.Distance)
		if err != nil {
			fmt.Printf("Failed to scan row in SeachData: %v\n", err.Error())
			continue
		}
		returnData = append(returnData, docChunk)
	}
	// fmt.Printf("Data %v\n", strconv.Itoa(100-int(returnData[0].Distance*100)))
	channel <- returnData
}
