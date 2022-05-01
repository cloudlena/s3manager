package s3manager_test

import "errors"

var (
	errS3                 = errors.New("mocked s3 error")
	errBucketDoesNotExist = errors.New("error: The specified bucket does not exist")
)
