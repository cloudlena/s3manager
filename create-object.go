package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// CreateObjectHandler allows to upload a new object
func CreateObjectHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			handleHTTPError(w, http.StatusUnprocessableEntity, err)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer func() {
			err = file.Close()
			if err != nil {
				log.Fatalln(err)
			}
		}()

		_, err = s3.PutObject(vars["bucketName"], handler.Filename, file, contentTypeOctetStream)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}
