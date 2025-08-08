package main

import (
	"datastar-placestovisit/models"
	"datastar-placestovisit/services"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	file, err := os.Open("worldcities.csv")
	if err != nil {
		fmt.Printf("Error opening file %v\n", err.Error())
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading records %v\n", err.Error())
		return
	}
	dataForDb := make([]models.WorldCities, len(records)-1)
	for idx, record := range records {
		if idx == 0 {
			continue
		}
		if len(record) > 10 {
			recordForDb := models.WorldCities{
				City:    strings.TrimSpace(record[0]),
				Lat:     strings.TrimSpace(record[2]),
				Lng:     strings.TrimSpace(record[3]),
				Country: strings.TrimSpace(record[4]),
				State:   strings.TrimSpace(record[7]),
			}
			dataForDb[idx-1] = recordForDb
		} else {
			fmt.Printf("%v record does not have all fields \n", record)
		}
	}
	noOfItemsPerTransaction := len(dataForDb) / 10

	dataForDb1 := dataForDb[:noOfItemsPerTransaction]
	dataForDb2 := dataForDb[noOfItemsPerTransaction : noOfItemsPerTransaction*2]
	dataForDb3 := dataForDb[noOfItemsPerTransaction*2 : noOfItemsPerTransaction*3]
	dataForDb4 := dataForDb[noOfItemsPerTransaction*3 : noOfItemsPerTransaction*4]
	dataForDb5 := dataForDb[noOfItemsPerTransaction*4 : noOfItemsPerTransaction*5]
	dataForDb6 := dataForDb[noOfItemsPerTransaction*5 : noOfItemsPerTransaction*6]
	dataForDb7 := dataForDb[noOfItemsPerTransaction*6 : noOfItemsPerTransaction*7]
	dataForDb8 := dataForDb[noOfItemsPerTransaction*7 : noOfItemsPerTransaction*8]
	dataForDb9 := dataForDb[noOfItemsPerTransaction*8 : noOfItemsPerTransaction*9]
	dataForDb10 := dataForDb[noOfItemsPerTransaction*9:]

	BulkInsert(dataForDb1)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb2)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb3)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb4)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb5)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb6)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb7)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb8)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb9)
	time.Sleep(10 * time.Second)
	BulkInsert(dataForDb10)

}

func BulkInsert(data []models.WorldCities) {
	channelForInsert := make(chan string)
	defer close(channelForInsert)
	go services.InsertMultipleWorldCity(data, channelForInsert)
	fmt.Println(<-channelForInsert)
}
