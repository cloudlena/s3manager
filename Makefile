.PHONY: all lint test build-docker deploy-cf clean

all:
	go build -o bin/s3manager ./cmd/s3manager

lint:
	golangci-lint run

test:
	go test -race -cover ./...

build-docker:
	docker build -t s3manager .

deploy-cf:
	GOOS=linux go build -ldflags="-s -w" -o bin/s3manager ./cmd/s3manager
	cf push -f deployments/cf/manifest.yml

clean:
	rm -rf bin/*
