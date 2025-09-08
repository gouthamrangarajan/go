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

func GetCachedTickerData(key models.CacheKey, channel chan<- []models.CacheData) {
	ctx := context.Background()
	response := []models.CacheData{}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	result, err := rdb.Get(ctx, key.Ticker+"_"+key.Date).Result()

	if err != nil {
		fmt.Printf("Error fetching %s data from Redis for date %s:%s\n", key.Ticker, key.Date, err)
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
		fmt.Printf("Cached data for %s found for date %s\n", key.Ticker, key.Date)
	}
	channel <- response
}

func SetCacheTickerData(key models.CacheKey, data []models.CacheData, channel chan<- string) {
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

	err = rdb.Set(ctx, key.Ticker+"_"+key.Date, dataJSON, 48*time.Hour).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for %s & date %s\n", key.Ticker, key.Date)
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

func CacheRefreshToken(tokens models.Tokens, channel chan<- string) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Username: os.Getenv("REDIS_USERNAME"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	err := rdb.Set(ctx, tokens.IdToken, tokens.RefreshToken, 24*time.Hour).Err()
	if err != nil {
		fmt.Println("Error setting data in Redis:", err)
		channel <- "ERROR"
		return
	}
	fmt.Printf("Successfully cached data for refresh token for id token\n")
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
