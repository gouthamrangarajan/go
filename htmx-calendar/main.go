package main

import (
	"fmt"
	"net/http"
)

func main() {

	http.HandleFunc("/", MainPage)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
