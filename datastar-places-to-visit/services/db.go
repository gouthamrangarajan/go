package services

import (
	"database/sql"
	"datastar-placestovisit/models"
	"fmt"
	"os"
	"strconv"
	"time"

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
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- response
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.WorldCities

		if err := rows.Scan(&item.Id, &item.City, &item.State, &item.Country, &item.Lat, &item.Lng); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			response = append(response, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- response
}

func InsertMultipleSpot(spots []models.TourismSpots, nearCityLatLng models.CityLatLng, channel chan string) {
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

	statement, err := transaction.Prepare("INSERT INTO spots (name,description,lat,lng,near_city,near_lat,near_lng,added) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		fmt.Printf("Unable to prepare db transaction statement %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	defer statement.Close()

	for _, row := range spots {
		fmt.Printf("Inserting %v data\n", row.Name)
		_, err := statement.Exec(row.Name, row.Description, row.Lat, row.Lng, nearCityLatLng.City, nearCityLatLng.Lat, nearCityLatLng.Lng, time.Now().Unix())
		if err != nil {
			// Handle error
			fmt.Printf("Error inserting %v into spots %v\n", row.Name, err.Error())
			success = "PARTIAL"
		}
	}

	err = transaction.Commit()

	if err != nil {
		fmt.Printf("Unable to commit transaction for inserting spots %v\n", err.Error())
		channel <- "ERROR"
		return
	}
	channel <- success
}
func InsertSpot(spot models.TourismSpots, nearCityLatLng models.CityLatLng, channel chan int) {
	id := 0
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- id
		return
	}
	defer db.Close()

	defer db.Close()
	result, err := db.Exec("INSERT INTO spots (name,description,lat,lng,near_city,near_lat,near_lng,added) VALUES (?,?,?,?,?,?,?,?)", spot.Name, spot.Description, spot.Lat, spot.Lng, nearCityLatLng.City, nearCityLatLng.Lat, nearCityLatLng.Lng, time.Now().Unix())
	if err != nil {
		fmt.Printf("Unable to execute insert spot  %v\n", err.Error())
		channel <- 0
		return
	}
	lastInsertedId, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Unable to get last inserted id after insert spot %v\n", err.Error())
		channel <- 0
		return
	}
	channel <- int(lastInsertedId)
}

func GetSpots(cityLatLng models.CityLatLng, limit int, channel chan []models.TourismSpots) {
	var response []models.TourismSpots
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- response
		return
	}
	defer db.Close()
	rows, err := db.Query("SELECT id,name,description,lat,lng,near_city,near_lat,near_lng,added FROM spots WHERE near_lat = ? AND near_lng = ? AND active=1 LIMIT ?", cityLatLng.Lat, cityLatLng.Lng, strconv.Itoa(limit))
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- response
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.TourismSpots
		if err := rows.Scan(&item.Id, &item.Name, &item.Description, &item.Lat, &item.Lng, &item.NearCity, &item.NearLat, &item.NearLng, &item.UnixTime); err != nil {
			fmt.Printf("Error scanning row:%v\n", err.Error())
		} else {
			response = append(response, item)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Error during rows iteration:%v\n", err.Error())
	}
	channel <- response
}

func InactivateSpots(cityLatLng models.CityLatLng, channel chan int) {
	db, err := getDb()
	if err != nil {
		fmt.Printf("Unable to get db %v\n", err.Error())
		channel <- 0
		return
	}
	defer db.Close()
	result, err := db.Exec("UPDATE spots SET active=0 WHERE near_lat = ? AND near_lng = ?", cityLatLng.Lat, cityLatLng.Lng)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err.Error())
		channel <- 0
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Unable to get rows affected after inactivate spots %v\n", err.Error())
		channel <- 0
		return
	}
	if rowsAffected <= 0 {
		fmt.Printf("No rows were updated during inactivate spots for lat %v lng %v\n", cityLatLng.Lat, cityLatLng.Lng)
	}
	channel <- int(rowsAffected)
}
