all:
	go build ./cmd/s3manager

lint:
	gometalinter ./... --vendor --tests

test:
	go test ./... -race -cover

build-docker:
	docker build . -f build/docker/Dockerfile -t s3manager

deploy-cf:
	GOOS=linux GOARCH=amd64 go build ./cmd/s3manager
	cf push -f deployments/cf/manifest.yml
