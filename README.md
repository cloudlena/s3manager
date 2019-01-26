# S3 Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/mastertinner/s3manager?style=flat-square)](https://goreportcard.com/report/github.com/mastertinner/s3manager)
[![Build Status](https://img.shields.io/travis/mastertinner/s3manager.svg?style=flat-square)](https://travis-ci.org/mastertinner/s3manager)
[![Docker Build](https://img.shields.io/docker/build/mastertinner/s3manager.svg?style=flat-square)](https://hub.docker.com/r/mastertinner/s3manager)

A Web GUI written in Go to manage S3 buckets from any provider.

## Build and Run Locally

1.  Run `make`
1.  Execute the created binary and visit <http://localhost:8080>

## Run Tests

1.  Run `make test`

## Build Docker Image

The image is available on [Docker Hub](https://hub.docker.com/r/mastertinner/s3manager/)

1.  Run `make build-docker`

## Run on Cloud Foundry

1.  Modify `deployments/cf/*` to your liking
1.  Run `make deploy-cf`
