package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"datastar-grocery/models"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"golang.org/x/exp/slices"
)

var dbPool *sql.DB

func InitDB() {
	dbUrl := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")
	url := fmt.Sprintf("%v?authToken=%v", dbUrl, authToken)

	var err error
	dbPool, err = sql.Open("libsql", url)
	if err != nil {
		log.Fatalf("Critical: Could not open DB connection: %v", err)
	}

	// NEW: Verify the connection is actually working
	err = dbPool.Ping()
	if err != nil {
		log.Fatalf("Critical: Could not connect to Turso (Ping failed): %v", err)
	}

	dbPool.SetMaxOpenConns(25)
	dbPool.SetMaxIdleConns(10)
}

func GetGroceryList(dbUrl string, authToken string, sort string) []models.Grocery {
	sort = strings.Trim(strings.ToUpper(sort), "")
	if !slices.Contains([]string{"ASC", "DESC"}, sort) {
		sort = " ORDER BY id DESC"
	} else {
		sort = fmt.Sprintf(" ORDER BY description COLLATE NOCASE %v", sort)
	}

	var data []models.Grocery = []models.Grocery{}
	query := "SELECT id,description,quantity,completed FROM grocery WHERE active = true" + sort
	rows, err := dbPool.Query(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		return data
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Grocery

		if err := rows.Scan(&item.Id, &item.Description, &item.Quantity, &item.Completed); err != nil {
			fmt.Println("Error scanning row:", err)
		} else {
			data = append(data, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
	}
	return data
}

func GetGroceryListViaChannel(databaseUrl string, authToken string, sort string, channel chan<- []models.Grocery) {
	channel <- GetGroceryList(databaseUrl, authToken, sort)
}

func GetGroceryListItem(dbUrl string, authToken string, id int) models.Grocery {
	var data models.Grocery = models.Grocery{}
	rows, err := dbPool.Query("SELECT id,description,quantity,completed FROM grocery WHERE active = true AND id = ?", id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		return data
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Grocery

		if err := rows.Scan(&item.Id, &item.Description, &item.Quantity, &item.Completed); err != nil {
			fmt.Println("Error scanning row:", err)
		} else {
			data = item
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
	}
	return data
}

func GetGroceryListItemViaChannel(databaseUrl string, authToken string, id int, channel chan<- models.Grocery) {
	channel <- GetGroceryListItem(databaseUrl, authToken, id)
}

func InsertGroceryItem(dbUrl string, authToken string, description string, quantity int) int {
	result, err := dbPool.Exec("INSERT INTO grocery (description, quantity) VALUES (?, ?)", description, quantity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		return 0
	}
	newId, errInsertId := result.LastInsertId()
	if errInsertId != nil {
		fmt.Fprintf(os.Stderr, "Error getting last inserted id: %v\n", errInsertId)
		return 0
	}
	return int(newId)
}

func InsertGroceryItemViaChannel(databaseUrl string, authToken string, name string, quantity int, channel chan<- int) {
	channel <- InsertGroceryItem(databaseUrl, authToken, name, quantity)
}

func DeleteGroceryItem(dbUrl string, authToken string, id int) int {
	result, err := dbPool.Exec("UPDATE grocery SET active = false WHERE id = ?", id)
	// result, err := db.Exec("DELETE FROM grocery WHERE id = ?", id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		return 0
	}
	rowsAffected, errDelete := result.RowsAffected()
	if errDelete != nil {
		fmt.Fprintf(os.Stderr, "Failed to get rows affected: %v\n", err)
		return 0
	}
	return int(rowsAffected)
}

func DeleteGroceryItemViaChannel(databaseUrl string, authToken string, id int, channel chan<- int) {
	channel <- DeleteGroceryItem(databaseUrl, authToken, id)
}

func UpdateQuantityGroceryItem(dbUrl string, authToken string, id int, quantity int) int {
	result, err := dbPool.Exec("UPDATE grocery SET quantity = ? WHERE id = ?", quantity, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		return 0
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Fprintf(os.Stderr, "Failed to get rows affected: %v\n", err)
		return 0
	}
	return int(rowsAffected)
}

func UpdateQuantityGroceryItemViaChannel(databaseUrl string, authToken string, id int, quantity int, channel chan<- int) {
	channel <- UpdateQuantityGroceryItem(databaseUrl, authToken, id, quantity)
}

func UpdateCompletedFieldGroceryItem(dbUrl string, authToken string, id int, completed bool) int {
	result, err := dbPool.Exec("UPDATE grocery SET completed = ? WHERE id = ?", completed, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute query: %v\n", err)
		return 0
	}
	rowsAffected, errUpdate := result.RowsAffected()
	if errUpdate != nil {
		fmt.Fprintf(os.Stderr, "Failed to get rows affected: %v\n", err)
		return 0
	}
	return int(rowsAffected)
}

func UpdateCompletedFieldGroceryItemViaChannel(databaseUrl string, authToken string, id int, completed bool, channel chan<- int) {
	channel <- UpdateCompletedFieldGroceryItem(databaseUrl, authToken, id, completed)
}
