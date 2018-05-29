.PHONY: all lint test build-docker deploy-cf

all:
	go build ./cmd/s3manager

lint:
	golangci-lint run --tests

test:
	go test -race -cover ./...

build-docker:
	docker build -f build/docker/Dockerfile -t s3manager .

deploy-cf:
	GOOS=linux go build -ldflags="-s -w" ./cmd/s3manager
	cf push -f deployments/cf/manifest.yml
