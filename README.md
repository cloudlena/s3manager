# S3 Manager

[![Build Status](https://travis-ci.org/mastertinner/s3-manager.svg?branch=master)](https://travis-ci.org/mastertinner/s3-manager)

A Web GUI written in Go to manage S3 buckets from any provider.

## Environment Variables

* `S3_ACCESS_KEY_ID`: Required. Your S3 access key ID
* `S3_SECRET_ACCESS_KEY`: Required. Your S3 secret access key
* `S3_ENDPOINT`: Optional. In case you are using a different S3 provider than AWS. Defaults to `s3.amazonaws.com`
* `V2_SIGNING`: Optional. In case your S3 provider still uses V2 Signing, set this to `true`

## Run locally

1. Run `go build`
1. Set environment variables in your env
1. Execute the binary and visit <http://localhost:8080>

## Run with Docker

1. Set environment variables in `docker-compose.yml`
1. Run `docker-compose up`
1. Visit <http://localhost:8080>

## Build with Docker and run anywhere

1. Run `docker run --rm -v "${PWD}:/go/src/github.com/mastertinner/s3-manager" -w /go/src/github.com/mastertinner/s3-manager golang curl https://glide.sh/get | sh && glide install && go build`

    To cross-compile for windows, use the `-e "GOOS=windows" -e "GOARCH=amd64"` flags (depending on your system, you might have to adjust `GOARCH`)

    To cross-compile for macOS, use the `-e "GOOS=darwin" -e "GOARCH=amd64"` flags (depending on your system, you might have to adjust `GOARCH`)

## Run on Cloud Foundry

1. Set environment variables in `manifest.yml`
1. Set host that isn't taken yet in `manifest.yml`
1. Run `cf push`
