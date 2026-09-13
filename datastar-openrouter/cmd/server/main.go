package main

import (
	"datastar-openrouter/internal/server"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	} else {
		fmt.Println("Loaded .env file successfully")
	}
	routerHandler := server.NewRouter().HttpHandler()
	http.ListenAndServe(":3000", routerHandler)
}
