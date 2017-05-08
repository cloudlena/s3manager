package s3manager

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// CreateObjectHandler uploads a new object.
func CreateObjectHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		bucketName := mux.Vars(r)["bucketName"]
		_, err = s3.PutObject(bucketName, handler.Filename, file, contentTypeOctetStream)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}
