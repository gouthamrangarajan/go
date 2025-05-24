package main

import (
	"encoding/json"
	"fmt"
	"htmx-calendar/models"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file")
	}
	channel := make(chan []models.EventData)
	defer close(channel)
	today := time.Now().Format("2006-01-02")
	userId := os.Getenv("SUPABASE_USER_ID")
	go GetCurrentDayDataForUser(userId, []string{today}, channel)
	response := <-channel
	fmt.Printf("Response: %v\n", response)
}
func GetCurrentDayDataForUser(userId string, dateRange []string, channel chan<- []models.EventData) {
	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
	apiUrl := os.Getenv("SUPABASE_API_URL")
	response := []models.EventData{}
	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
	if err != nil {
		fmt.Printf("Error connecting to supabase %v\n", err.Error())
		channel <- response
		return
	}
	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).In("date", dateRange).Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()
	if err != nil {
		fmt.Printf("Error executing query %v\n", err.Error())
		channel <- response
		return
	}
	if err := json.Unmarshal(data, &response); err != nil {
		fmt.Printf("Error unmarshalling results %v\n", err.Error())
	}
	channel <- response
}

// func GetWeeklyDataForUserWithExactDate(userId string, dateRange []string, channel chan<- []models.EventData) {
// 	serviceRole := os.Getenv("SUPABASE_SERVICE_ROLE")
// 	apiUrl := os.Getenv("SUPABASE_API_URL")
// 	response := []models.EventData{}
// 	client, err := supabase.NewClient(apiUrl, serviceRole, &supabase.ClientOptions{})
// 	if err != nil {
// 		fmt.Printf("Error connecting to supabase %v\n", err.Error())
// 		channel <- response
// 		return
// 	}
// 	data, _, err := client.From("calendar").Select("id, task, frequency, date, stopAfter, exact", "exact", false).Eq("user_id", userId).In("date", dateRange).Order("created_at", &postgrest.OrderOpts{Ascending: true}).Execute()
// 	if err != nil {
// 		fmt.Printf("Error executing query %v\n", err.Error())
// 		channel <- response
// 		return
// 	}
// 	if err := json.Unmarshal(data, &response); err != nil {
// 		fmt.Printf("Error unmarshalling results %v\n", err.Error())
// 	}
// 	channel <- response
// }
