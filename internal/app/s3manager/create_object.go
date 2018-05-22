package s3manager

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// CreateObjectHandler uploads a new object.
func CreateObjectHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error parsing multipart form"))
			return
		}
		file, handler, err := r.FormFile("file")
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error getting file from form"))
			return
		}
		defer func() {
			err = file.Close()
			if err != nil {
				log.Fatalln(errors.Wrap(err, "error closing file"))
			}
		}()

		_, err = s3.PutObject(bucketName, handler.Filename, file, 1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error putting object"))
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}
