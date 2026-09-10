package main

import (
	"datastar-openrouter-whisper101/internal/server"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Error loading .env file")
	}
	router := server.NewRouter()
	http.ListenAndServe(":3000", router)

}
