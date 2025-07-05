package services

import (
	"context"
	"datastar-stock/models"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func GetCachedData(ticker string, channel chan<- []models.CacheData) {
	response := []models.CacheData{}
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, ticker+"_"+time.Now().Format("2006-01-02")).Result()

	if err != nil {
		fmt.Printf("Error fetching %s data from Redis:%s\n", ticker, err)
		channel <- response
		return
	}
	// fmt.Println("Fetched data from Redis:", result)
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		fmt.Println("Error unmarshalling data from Redis:", err)
		channel <- response
		return
	}
	if len(response) > 0 {
		fmt.Printf("Cached data for %s found\n", ticker)
	}
	channel <- response
}

func SetCachedData(ticker string, data []models.CacheData, channel chan<- string) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	dataJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling data to JSON:", err)
		channel <- "ERROR"
		return
	}

	err = rdb.Set(ctx, ticker+"_"+time.Now().Format("2006-01-02"), dataJSON, 48*time.Hour).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for %s\n", ticker)
	channel <- "OK"
}
