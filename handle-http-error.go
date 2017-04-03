package main

import (
	"log"
	"net/http"
)

// handleHTTPError handles HTTP errors
func handleHTTPError(w http.ResponseWriter, msg string, err error, statusCode int) {
	http.Error(w, msg, statusCode)
	if err != nil {
		log.Println(msg+":", err.Error())
	} else {
		log.Println(msg)
	}
}
