# S3 Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/cloudlena/s3manager)](https://goreportcard.com/report/github.com/cloudlena/s3manager)
[![Build Status](https://github.com/cloudlena/s3manager/actions/workflows/main.yml/badge.svg)](https://github.com/cloudlena/s3manager/actions)

A Web GUI written in Go to manage S3 buckets from any provider.

![Screenshot](https://raw.githubusercontent.com/cloudlena/s3manager/main/screenshot.png)

## Features

- List all buckets in your account
- Create a new bucket
- List all objects in a bucket
- Upload new objects to a bucket
- Download object from a bucket
- Delete an object in a bucket

## Usage

### Configuration

The application can be configured with the following environment variables:

- `ENDPOINT`: The endpoint of your S3 server (defaults to `s3.amazonaws.com`)
- `REGION`: The region of your S3 server (defaults to `""`)
- `ACCESS_KEY_ID`: Your S3 access key ID (required) (works only if `USE_IAM` is `false`)
- `SECRET_ACCESS_KEY`: Your S3 secret access key (required) (works only if `USE_IAM` is `false`)
- `USE_SSL`: Whether your S3 server uses SSL or not (defaults to `true`)
- `SKIP_SSL_VERIFICATION`: Whether the HTTP client should skip SSL verification (defaults to `false`)
- `SIGNATURE_TYPE`: The signature type to be used (defaults to `V4`; valid values are `V2, V4, V4Streaming, Anonymous`)
- `PORT`: The port the app should listen on (defaults to `8080`)
- `ALLOW_DELETE`: Enable buttons to delete objects (defaults to `true`)
- `FORCE_DOWNLOAD`: Add response headers for object downloading instead of opening in a new tab (defaults to `true`)
- `LIST_RECURSIVE`: List all objects in buckets recursively (defaults to `false`)
- `USE_IAM`: Use IAM role instead of key pair (defaults to `false`)
- `IAM_ENDPOINT`: Endpoint for IAM role retrieving (Can be blank for AWS)
- `SSE_TYPE`: Specified server side encryption (defaults blank) Valid values can be `SSE`, `KMS`, `SSE-C` all others values don't enable the SSE
- `SSE_KEY`: The key needed for SSE method (only for `KMS` and `SSE-C`)
- `TIMEOUT`: The read and write timeout in seconds (default to `600` - 10 minutes)
- `NAVBAR_COLOR`: The background color for the navigation bar (defaults to `#ee6e73` - Materialize teal color)
- `LOGO_PATH`: Custom path to logo image (defaults to `/static/img/logo.png` if the file exists, empty otherwise). When a logo is provided, both the logo and "S3 Manager" text are displayed together in the navbar.
- `BUTTON_COLOR`: The background color for primary buttons (defaults to `#f44336` - Materialize red color)

#### UI Customization Examples

Customize the navbar color to match your brand:
```bash
export NAVBAR_COLOR="#2196F3"  # Blue navbar
export NAVBAR_COLOR="#4CAF50"  # Green navbar
export NAVBAR_COLOR="#FF9800"  # Orange navbar
```

Customize button colors:
```bash
export BUTTON_COLOR="#2196F3"  # Blue buttons
export BUTTON_COLOR="#4CAF50"  # Green buttons
export BUTTON_COLOR="#FF9800"  # Orange buttons
```

Use a custom logo (must be accessible as a static file):
```bash
export LOGO_PATH="/static/img/company-logo.png"
```

The application automatically detects if `web/static/img/logo.png` exists and uses it as the default logo.

#### Custom Styling

The application uses CSS custom properties (CSS variables) for dynamic theming. All styling is properly separated into CSS files rather than inline styles for better maintainability. The main custom styles are located in `/static/css/s3manager.css`.

### Build and Run Locally

1.  Run `make build`
1.  Execute the created binary and visit <http://localhost:8080>

### Run Container image

1. Run `docker run -p 8080:8080 -e 'ACCESS_KEY_ID=XXX' -e 'SECRET_ACCESS_KEY=xxx' cloudlena/s3manager`

### Deploy to Kubernetes

You can deploy S3 Manager to a Kubernetes cluster using the [Helm chart](https://github.com/sergeyshevch/s3manager-helm).

## Development

### Lint Code

1. Run `make lint`

### Run Tests

1.  Run `make test`

### Build Container Image

The image is available on [Docker Hub](https://hub.docker.com/r/cloudlena/s3manager/).

1.  Run `make build-image`

### Run Locally for Testing

There is an example [docker-compose.yml](https://github.com/cloudlena/s3manager/blob/main/docker-compose.yml) file that spins up a S3 service and the S3 Manager. You can try it by issuing the following command:

```shell
$ docker-compose up
```

## GitHub Stars

[![GitHub stars over time](https://starchart.cc/cloudlena/s3manager.svg?variant=adaptive)](https://starchart.cc/cloudlena/s3manager)
