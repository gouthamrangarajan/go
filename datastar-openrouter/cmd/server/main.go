package main

import (
	"datastar-openrouter/internal/server"
	"datastar-openrouter/services"
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
	services.InitDB()
	routerHandler := server.NewRouter().NewHttpHandler()
	http.ListenAndServe(":3000", routerHandler)
}
