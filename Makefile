all:
	go build ./cmd/s3manager

test:
	go test

build-docker:
	docker run --rm -v "${PWD}:/go/src/github.com/mastertinner/s3manager" -w /go/src/github.com/mastertinner/s3manager golang go build ./cmd/s3manager

deploy-cf:
	GOOS=linux GOARCH=amd64 go build ./cmd/s3manager
	cf push
