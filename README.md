# S3 Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/mastertinner/s3manager?style=flat-square)](https://goreportcard.com/report/github.com/mastertinner/s3manager)
[![Build Status](https://travis-ci.org/mastertinner/s3manager.svg?branch=master&style=flat-square)](https://travis-ci.org/mastertinner/s3manager)
[![Release](https://img.shields.io/github/release/mastertinner/s3manager.svg?style=flat-square)](https://github.com/mastertinner/s3manager/releases/latest)

A Web GUI written in Go to manage S3 buckets from any provider.

## Install Dependencies

1.  Install [vgo](https://github.com/golang/vgo)
1.  Run `vgo mod vendor`

## Build and Run Locally

1.  Run `make`
1.  Execute the created binary and visit <http://localhost:8080>

## Run Tests

1.  Run `make test`

## Build Docker Image

1.  Run `make build-docker`

## Run on Cloud Foundry

1.  Modify `deployments/cf/*` to your liking
1.  Run `make deploy-cf`
