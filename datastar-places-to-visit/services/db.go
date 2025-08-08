package services

import (
	"database/sql"
	"datastar-placestovisit/models"
	"fmt"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func getDb() (*sql.DB, error) {
	url := os.Getenv("TURSO_URL") + "?authToken=" + os.Getenv("TURSO_AUTH_TOKEN")
	db, err := sql.Open("libsql", url)
	return db, err
}

func InsertMultipleWorldCity(worldCities []models.WorldCities, channel chan string) {
	success := "SUCCESS"
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	defer db.Close()

	transaction, err := db.Begin()
	if err != nil {
		fmt.Printf("Unable to get db transaction %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare("INSERT INTO world_cities (city,state,country,lat,lng) VALUES (?,?,?,?,?)")
	if err != nil {
		fmt.Printf("Unable to prepare db transaction statement %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	defer statement.Close()

	for _, row := range worldCities {
		fmt.Printf("Inserting %v data\n", row.City)
		_, err := statement.Exec(row.City, row.State, row.Country, row.Lat, row.Lng)
		if err != nil {
			// Handle error
			fmt.Printf("Error inserting %v %v %v %v %v into world_cities %v\n", row.City, row.State, row.Country, row.Lat, row.Lng, err.Error())
			success = "PARTIAL"
		}
	}

	err = transaction.Commit()

	if err != nil {
		fmt.Printf("Unable to commit transaction for inserting world cities %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	channel <- success
}

func InsertWorldCity(worldCity models.WorldCities, channel chan int) {
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("INSERT INTO world_cities (city,state,country,lat,lng) VALUES (?,?,?,?,?)", worldCity.City, worldCity.State, worldCity.Country, worldCity.Lat, worldCity.Lng)
	if err != nil {
		fmt.Printf("Unable to execute insert world city %v\n", err.Error())
		channel <- 0
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Unable to get rows affected after insert world city %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(rowsAffected)
}

func SearchWorldCity(search string, channel chan []models.WorldCities) {
	searchString := "%" + search + "%"
	var response []models.WorldCities
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- response
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT id,city,state,country,lat,lng FROM world_cities WHERE city LIKE ? OR state LIKE ? or country LIKE ? LIMIT 25", searchString, searchString, searchString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		channel <- response
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.WorldCities

		if err := rows.Scan(&item.Id, &item.City, &item.State, &item.Country, &item.Lat, &item.Lng); err != nil {
			fmt.Println("Error scanning row:", err)
		} else {
			response = append(response, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error during rows iteration:", err)
	}
	channel <- response
}
