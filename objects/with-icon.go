package objects

import minio "github.com/minio/minio-go"

// WithIcon is a minio object with an added icon
type WithIcon struct {
	minio.ObjectInfo
	Icon string
}
