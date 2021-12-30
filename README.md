# S3 Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/cloudlena/s3manager?style=flat-square)](https://goreportcard.com/report/github.com/cloudlena/s3manager)
[![Build Status](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Factions-badge.atrox.dev%2Fcloudlena%2Fs3manager%2Fbadge&style=flat-square)](https://github.com/cloudlena/s3manager/actions)

A Web GUI written in Go to manage S3 buckets from any provider.

![Screenshot](https://raw.githubusercontent.com/cloudlena/s3manager/master/screenshot.png)

## Usage

### Configuration

The application can be configured with the following environment variables:

- `ENDPOINT`: The endpoint of your S3 server (defaults to `s3.amazonaws.com`)
- `REGION`: The region of your S3 server (defaults to `""`)
- `ACCESS_KEY_ID`: Your S3 access key ID (required)
- `SECRET_ACCESS_KEY`: Your S3 secret access key (required)
- `USE_SSL`: Whether your S3 server uses SSL or not (defaults to `true`)
- `SKIP_SSL_VERIFICATION`: Whether the HTTP client should skip SSL verification (defaults to `false`)
- `PORT`: The port the s3manager app should listen on (defaults to `8080`)

### Build and Run Locally

1.  Run `make build`
1.  Execute the created binary and visit <http://localhost:8080>

### Run Container image

1. Run `docker run -p 8080:8080 -e 'ACCESS_KEY_ID=XXX' -e 'SECRET_ACCESS_KEY=xxx' mastertinner/s3manager`

## Development

### Lint Code

1. Run `make lint`

### Run Tests

1.  Run `make test`

### Build Container Image

The image is available on [Docker Hub](https://hub.docker.com/r/mastertinner/s3manager/)

1.  Run `make build-image`

### Run Locally for Testing

There is an example [docker-compose.yml](https://github.com/cloudlena/s3manager/blob/master/docker-compose.yml) file that spins up an S3 service and the s3manager. You can try it by issuing the following command:

```shell
$ docker-compose up
```
