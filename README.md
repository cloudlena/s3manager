# S3 Manager

[![Build Status](https://travis-ci.org/mastertinner/s3manager.svg?branch=master)](https://travis-ci.org/mastertinner/s3manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/mastertinner/s3manager)](https://goreportcard.com/report/github.com/mastertinner/s3manager)

A Web GUI written in Go to manage S3 buckets from any provider.

## Run locally

1. Run `make`
1. Execute the created binary and visit <http://localhost:8080>

## Run with Docker

1. Set environment variables in `docker-compose.yml`
1. Run `docker-compose up`
1. Visit <http://localhost:8080>

## Build with Docker and run anywhere

1. Run `make build-docker`

    To cross-compile for windows, add the `-e "GOOS=windows" -e "GOARCH=amd64"` flags to the `Makefile` (depending on your system, you might have to adjust `GOARCH`)

    To cross-compile for macOS, add the `-e "GOOS=darwin" -e "GOARCH=amd64"` flags to the `Makefile` (depending on your system, you might have to adjust `GOARCH`)

## Run on Cloud Foundry

1. Change the service in `manifest.yml` to represent your S3 service (if you are using an external S3 provider, you'll have to switch the service type in `entrypoint-cf.sh` from `dynstrg` to `user-provided` and create the respective user-provided service with `cf create-user-provided-service`)
1. Change `host` in `manifest.yml` to something that isn't taken yet
1. Run `make deploy-cf`
