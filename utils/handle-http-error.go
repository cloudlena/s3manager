package utils

import (
	"log"
	"net/http"
)

// HandleHTTPError handles HTTP errors
func HandleHTTPError(w http.ResponseWriter, msg string, err error, statusCode int) {
	http.Error(w, msg, statusCode)
	if err != nil {
		log.Println(msg+":", err.Error())
	} else {
		log.Println(msg)
	}
}
