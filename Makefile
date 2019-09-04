.PHONY: build
build:
	go build -o bin/s3manager ./cmd/s3manager

.PHONY: run
run:
	go run cmd/s3manager/main.go

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -race -cover ./...

.PHONY: build-docker
build-docker:
	docker build -t s3manager .

.PHONY: deploy-cf
deploy-cf:
	GOOS=linux go build -ldflags="-s -w" -o bin/s3manager ./cmd/s3manager
	cf push -f deployments/cf/manifest.yml

.PHONY: clean
clean:
	rm -rf bin
