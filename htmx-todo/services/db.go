package services

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"htmx-todo/models"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"golang.org/x/exp/slices"
)

func createDb(dbUrl string, authToken string) *sql.DB {
	url := fmt.Sprintf("%v?authToken=%v", dbUrl, authToken)

	db, err := sql.Open("libsql", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", url, err)
		os.Exit(1)
	}
	// defer db.Close()
	return db
}

func GetGroceryList(dbUrl string, authToken string, sort string) []models.Grocery {
	sort = strings.Trim(strings.ToUpper(sort), "")
	if !slices.Contains([]string{"ASC", "DESC"}, sort) {
		sort = " ORDER BY id DESC"
	} else {
		sort = fmt.Sprintf(" ORDER BY description COLLATE NOCASE %v", sort)
	}
	db := createDb(dbUrl, authToken)
	defer db.Close()
	var data []models.Grocery = []models.Grocery{}
	query := "SELECT id,description,quantity,completed FROM grocery WHERE active = true" + sort
	rows, err := db.Query(query)
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

func GetGroceryListItem(dbUrl string, authToken string, id int) models.Grocery {
	db := createDb(dbUrl, authToken)
	defer db.Close()
	var data models.Grocery = models.Grocery{}
	rows, err := db.Query("SELECT id,description,quantity,completed FROM grocery WHERE active = true AND id = ?", id)
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

func InsertGroceryItem(dbUrl string, authToken string, description string, quantity int) int {
	db := createDb(dbUrl, authToken)
	defer db.Close()
	result, err := db.Exec("INSERT INTO grocery (description, quantity) VALUES (?, ?)", description, quantity)
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

func DeleteGroceryItem(dbUrl string, authToken string, id int) int {
	db := createDb(dbUrl, authToken)
	defer db.Close()
	// result, err := db.Exec("UPDATE grocery SET active = false WHERE id = ?", id)
	result, err := db.Exec("DELETE FROM grocery WHERE id = ?", id)
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

func UpdateQuantityGroceryItem(dbUrl string, authToken string, id int, quantity int) int {
	db := createDb(dbUrl, authToken)
	defer db.Close()
	result, err := db.Exec("UPDATE grocery SET quantity = ? WHERE id = ?", quantity, id)
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

func UpdateCompletedFieldGroceryItem(dbUrl string, authToken string, id int, completed bool) int {
	db := createDb(dbUrl, authToken)
	defer db.Close()
	result, err := db.Exec("UPDATE grocery SET completed = ? WHERE id = ?", completed, id)
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
