# S3 Manager

A Web GUI written in Go to manage S3 buckets from any provider.

## Environment Variables

* `S3_ACCESS_KEY_ID`: Required. Your S3 access key ID
* `S3_SECRET_ACCESS_KEY`: Required. Your S3 secret access key
* `S3_ENDPOINT`: Optional. In case you are using a different S3 provider than AWS. Defaults to `s3.amazonaws.com`
* `V2_SIGNING`: Optional. In case your S3 provider still uses V2 Signing, set this to `true`

## Run locally

1. Set all environment variables
1. Run `go build`
1. Run the binary and visit <http://localhost:8080>

## Push to Cloud Foundry

1. Set environment variables in `manifest.yml`
1. Set hostname that isn't taken yet in `manifest.yml`
1. Run `cf push`
