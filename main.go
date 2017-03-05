package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	router := NewRouter()

	log.Fatal(http.ListenAndServe(":"+port, router))
}
