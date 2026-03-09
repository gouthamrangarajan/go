package main

import (
	"datastar-portfolio/models"
	"datastar-portfolio/services"
	"encoding/json"
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
		fmt.Println("Loaded .env file successfully")
	}
	fmt.Printf("Job Started %v\n", time.Now())
	dataBytes, err := os.ReadFile("data.json")
	if err != nil {
		fmt.Printf("Error reading data.json: %v\n", err)
		return
	}
	var jsonData models.JsonData
	err = json.Unmarshal(dataBytes, &jsonData)
	if err != nil {
		fmt.Printf("Error unmarshalling data.json: %v\n", err)
		return
	}
	// fmt.Printf("Parsed JSON data successfully: %v\n", jsonData)
	dbDataToInsert := convertJsonDataToDbData(jsonData)
	channel := make(chan int)
	go services.InsertDemos(dbDataToInsert, channel)
	insertedCount := <-channel
	fmt.Printf("Inserted %v demo items into the database: %v\n", insertedCount, time.Now())

}

func convertJsonDataToDbData(jsonData models.JsonData) []models.DemoItem {
	var dbData []models.DemoItem
	for _, item := range jsonData.AllDemoItems {
		dbItem := models.DemoItem{
			Title:             item.Title,
			ImgSrc:            item.ImgSrc,
			Description:       item.Description,
			Url:               item.Url,
			Service:           item.Service,
			Tags:              strings.Join(item.Tags, ","),
			ImgBadgeLightMode: item.ImgBadgeLightMode,
			CodeUrl:           "",
			Display:           item.Display,
		}
		dbItem.ImgSrc = strings.ReplaceAll(dbItem.ImgSrc, "/imgs/cloud/", "/assets/images/")
		dbItem.ImgSrc = strings.ReplaceAll(dbItem.ImgSrc, "/imgs/github/", "/assets/images/")
		dbItem.ImgSrc = strings.ReplaceAll(dbItem.ImgSrc, "/imgs/codepen/", "/assets/images/")
		dbData = append(dbData, dbItem)
	}
	return dbData
}
