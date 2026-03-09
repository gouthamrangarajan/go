package services

import (
	"database/sql"
	"datastar-portfolio/models"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

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

func GetFeaturedDemos(channel chan<- []models.DemoItem) {
	var data []models.DemoItem = []models.DemoItem{}
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in GetFeaturedDemos: %v\n", err.Error())
		channel <- data
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT id,title,img_src,description,url,service,tags,img_badge_light_mode,code_url FROM demos WHERE is_featured=1 AND display=1 ORDER BY sort_order")
	if err != nil {
		fmt.Printf("Failed to execute query in GetFeaturedDemos: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.DemoItem

		if err := rows.Scan(&item.Id, &item.Title, &item.ImgSrc, &item.Description, &item.Url, &item.Service, &item.Tags, &item.ImgBadgeLightMode, &item.CodeUrl); err != nil {
			fmt.Printf("Error scanning row in GetFeaturedDemos:%v\n", err.Error())
		} else {
			data = append(data, item)
			// fmt.Printf("Fetched demo item: %v\n", item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetFeaturedDemos:%v\n", err.Error())
	}
	channel <- data
}

func InsertDemos(data []models.DemoItem, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in InsertDemos: %v\n", err.Error())
		channel <- 0
		return
	}
	defer db.Close()

	insertStatement := "INSERT INTO demos (title, img_src, description, url, service, tags, img_badge_light_mode, code_url,is_featured,sort_order,display) VALUES %s"
	valueStrings := make([]string, 0, len(data))
	valueArgs := make([]interface{}, 0, len(data)*11)

	for _, item := range data {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?,?,?)")
		valueArgs = append(valueArgs, item.Title, item.ImgSrc, item.Description, item.Url, item.Service, item.Tags, item.ImgBadgeLightMode, item.CodeUrl, 0, 0, item.Display)
	}
	query := fmt.Sprintf(insertStatement, strings.Join(valueStrings, ","))

	result, err := db.Exec(query, valueArgs...)
	if err != nil {
		fmt.Printf("Failed to execute insert query in InsertDemos: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected in InsertDemos: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func GetAllDemos(channel chan<- []models.DemoItem, isActiveOnly bool) {
	var data []models.DemoItem = []models.DemoItem{}
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in GetAllDemos: %v\n", err.Error())
		channel <- data
		return
	}
	defer db.Close()
	queryStatement := "SELECT id,title,img_src,description,url,service,tags,img_badge_light_mode,code_url FROM demos WHERE display=1 ORDER BY sort_order"
	if !isActiveOnly {
		queryStatement = "SELECT id,title,img_src,description,url,service,tags,img_badge_light_mode,code_url FROM demos ORDER BY sort_order"
	}
	rows, err := db.Query(queryStatement)
	if err != nil {
		fmt.Printf("Failed to execute query in GetAllDemos: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.DemoItem

		if err := rows.Scan(&item.Id, &item.Title, &item.ImgSrc, &item.Description, &item.Url, &item.Service, &item.Tags, &item.ImgBadgeLightMode, &item.CodeUrl); err != nil {
			fmt.Printf("Error scanning row in GetAllDemos:%v\n", err.Error())
		} else {
			data = append(data, item)
			// fmt.Printf("Fetched demo item: %v\n", item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetAllDemos:%v\n", err.Error())
	}
	channel <- data
}
func GetAllDemosWithServiceFilter(channel chan<- []models.DemoItem, services []string) {
	var data []models.DemoItem = []models.DemoItem{}
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in GetAllDemosWithServiceFilter: %v\n", err.Error())
		channel <- data
		return
	}
	defer db.Close()
	queryStatement := "SELECT id,title,img_src,description,url,service,tags,img_badge_light_mode,code_url FROM demos WHERE display=1"
	args := []interface{}{}
	if len(services) > 0 && !slices.Contains(services, "All") {
		servicePlaceholders := make([]string, len(services))
		for i := range services {
			servicePlaceholders[i] = "?"
			args = append(args, services[i])
		}
		queryStatement += fmt.Sprintf(" AND service IN (%s) ", strings.Join(servicePlaceholders, ","))

	}
	queryStatement += " ORDER BY sort_order"

	rows, err := db.Query(queryStatement, args...)
	if err != nil {
		fmt.Printf("Failed to execute query in GetAllDemosWithServiceFilter: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.DemoItem

		if err := rows.Scan(&item.Id, &item.Title, &item.ImgSrc, &item.Description, &item.Url, &item.Service, &item.Tags, &item.ImgBadgeLightMode, &item.CodeUrl); err != nil {
			fmt.Printf("Error scanning row in GetAllDemosWithServiceFilter:%v\n", err.Error())
		} else {
			data = append(data, item)
			// fmt.Printf("Fetched demo item: %v\n", item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetAllDemosWithServiceFilter:%v\n", err.Error())
	}
	channel <- data
}
func UpdateDemosEmbeddings(data []models.DemoItem, channel chan<- int) {
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in UpdateDemosEmbeddings: %v\n", err.Error())
		channel <- 0
		return
	}
	defer db.Close()

	updateStatement := "UPDATE demos SET embeddings = CASE id"
	params := []interface{}{}
	idStrings := make([]string, 0, len(data))

	for _, item := range data {
		idStrings = append(idStrings, fmt.Sprintf("%d", item.Id))
		embeddingBytes, err := json.Marshal(item.Embeddings)
		if err != nil {
			fmt.Printf("Failed to marshal embeddings for demo item with id %v: %v\n", item.Id, err.Error())
			continue
		}

		updateStatement += " WHEN ? THEN vector32(?)"
		params = append(params, item.Id, string(embeddingBytes))
	}
	updateStatement += " END WHERE id IN ("
	updateStatement += strings.Join(idStrings, ",") + ")"

	result, err := db.Exec(updateStatement, params...)
	if err != nil {
		fmt.Printf("Failed to execute update query in UpdateDemosEmbeddings: %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Failed to get rows affected in UpdateDemosEmbeddings: %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}
func SearchDemos(channel chan []models.DemoItem, embeddingToSearch []float32, services []string) {
	var data []models.DemoItem = []models.DemoItem{}
	db, err := createDb()
	if err != nil {
		fmt.Printf("Failed to create db connection in UpdateDemosEmbeddings: %v\n", err.Error())
		channel <- data
		return
	}
	defer db.Close()
	embeddingToSearchStrBytes, err := json.Marshal(embeddingToSearch)
	if err != nil {
		fmt.Printf("Failed to marshal embeddingToSearch in SearchDemos: %v\n", err.Error())
		channel <- data
		return
	}
	embeddingToSearchStr := string(embeddingToSearchStrBytes)
	args := []interface{}{embeddingToSearchStr}

	queryStatement := "SELECT id,title,img_src,description,url,service,tags,img_badge_light_mode,code_url FROM demos WHERE display=1 AND vector_distance_cos(embeddings, vector32(?)) < 0.8"

	if len(services) > 0 && !slices.Contains(services, "All") {
		servicePlaceholders := make([]string, len(services))
		for i := range services {
			servicePlaceholders[i] = "?"
			args = append(args, services[i])
		}
		queryStatement += fmt.Sprintf(" AND service IN (%s) ", strings.Join(servicePlaceholders, ","))

	}
	queryStatement += " ORDER BY vector_distance_cos(embeddings, vector32(?))"
	args = append(args, embeddingToSearchStr)
	// fmt.Printf("Executing query %s: args %s", queryStatement, args[1])
	rows, err := db.Query(queryStatement, args...)
	if err != nil {
		fmt.Printf("Failed to execute query in SearchDemos: %v\n", err.Error())
		channel <- data
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item models.DemoItem

		if err := rows.Scan(&item.Id, &item.Title, &item.ImgSrc, &item.Description, &item.Url, &item.Service, &item.Tags, &item.ImgBadgeLightMode, &item.CodeUrl); err != nil {
			fmt.Printf("Error scanning row in SearchDemos:%v\n", err.Error())
		} else {
			data = append(data, item)
		}
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration in GetAllDemos:%v\n", err.Error())
	}
	channel <- data
}
