.PHONY: build
build:
	go build -o bin/s3manager

.PHONY: run
run:
	go run

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -race -cover ./...

.PHONY: build-image
build-image:
	docker build -t s3manager .

.PHONY: build-multiarch-image
build-multiarch-image:
	docker buildx create --name multiarch-builder --use || docker buildx use multiarch-builder
	docker buildx build --platform linux/amd64,linux/arm64 -t dimuthnc/s3manager:latest --push .

.PHONY: clean
clean:
	rm -rf bin
