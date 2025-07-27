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

func GetCachedTickerData(ticker string, date string, channel chan<- []models.CacheData) {
	ctx := context.Background()
	response := []models.CacheData{}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, ticker+"_"+date).Result()

	if err != nil {
		fmt.Printf("Error fetching %s data from Redis for date %s:%s\n", ticker, date, err)
		channel <- response
		return
	}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		fmt.Println("Error unmarshalling data from Redis:", err)
		channel <- response
		return
	}
	if len(response) > 0 {
		fmt.Printf("Cached data for %s found for date %s\n", ticker, date)
	}
	channel <- response
}

func SetCacheTickerData(ticker string, date string, data []models.CacheData, channel chan<- string) {
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

	err = rdb.Set(ctx, ticker+"_"+date, dataJSON, 48*time.Hour).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for %s & date %s\n", ticker, date)
	channel <- "OK"
}

func GetCachedCompaniesData(channel chan<- []models.CompanyFromDb) {
	ctx := context.Background()
	response := []models.CompanyFromDb{}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, "companies").Result()

	if err != nil {
		fmt.Printf("Error fetching companies data from Redis  %s\n", err)
		channel <- response
		return
	}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		fmt.Println("Error unmarshalling data from Redis:", err)
		channel <- response
		return
	}
	if len(response) > 0 {
		fmt.Printf("Cached data for companies found\n")
	}
	channel <- response
}

func SetCacheCompaniesData(data []models.CompanyFromDb, channel chan<- string) {
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

	err = rdb.Set(ctx, "companies", dataJSON, 0).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for companies count :%v\n", len(data))
	channel <- "OK"
}

func GetCachedPopularsData(channel chan<- []string) {
	ctx := context.Background()
	response := []string{}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, "populars").Result()

	if err != nil {
		fmt.Printf("Error fetching populars data from Redis %v\n", err)
		channel <- response
		return
	}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		fmt.Println("Error unmarshalling data from Redis:", err)
		channel <- response
		return
	}
	if len(response) > 0 {
		fmt.Printf("Cached data for populars found\n")
	}
	channel <- response
}

func SetCachePopularsData(data []string, channel chan<- string) {
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

	err = rdb.Set(ctx, "populars", dataJSON, 0).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for populars\n")
	channel <- "OK"
}

func CacheRefreshToken(idToken string, refreshToken string, channel chan<- string) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	err := rdb.Set(ctx, idToken, refreshToken, 24*time.Hour).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for refresh token for id token %v\n", idToken)
	channel <- "OK"
}

func GetCachedRefreshToken(idToken string, channel chan<- string) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, idToken).Result()

	if err != nil {
		fmt.Printf("Error fetching refresh token from Redis  %s\n", err)
		channel <- ""
		return
	}
	channel <- result
}
