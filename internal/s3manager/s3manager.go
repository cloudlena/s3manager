// Package s3manager allows to interact with an S3 compatible storage.
package s3manager

type S3Manager struct {
	s3          S3
	allowDelete bool
	sseType     string
	sseKey      string
}

func New(s3 S3, allowDelete bool, sseType, sseKey string) *S3Manager {
	return &S3Manager{
		s3:          s3,
		allowDelete: allowDelete,
		sseType:     sseType,
		sseKey:      sseKey,
	}
}
