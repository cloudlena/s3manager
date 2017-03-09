package main

import (
	"log"
	"net/http"
	"os"

	minio "github.com/minio/minio-go"
)

// Server is a server containing a minio client
type Server struct {
	s3 *minio.Client
}

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	router := NewRouter()

	log.Fatal(http.ListenAndServe(":"+port, router))
}
