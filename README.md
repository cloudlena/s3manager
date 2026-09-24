# S3 Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/cloudlena/s3manager)](https://goreportcard.com/report/github.com/cloudlena/s3manager)
[![Build Status](https://github.com/cloudlena/s3manager/actions/workflows/main.yml/badge.svg)](https://github.com/cloudlena/s3manager/actions)

A Web GUI written in Go to manage S3 buckets from any provider.

![Screenshot](https://raw.githubusercontent.com/cloudlena/s3manager/main/screenshot.png)

## Features

- Manage several S3 accounts side by side and switch between them
- List, create and delete buckets
- View and edit a bucket's policy, including making a bucket publicly readable
- List a bucket's objects with search, sorting and pagination
- Upload single objects or whole folders to a bucket
- Download an object, or several selected ones as a ZIP archive
- Open an object in the browser (click its name or use the `Open` action)
- Delete a single object or several selected ones
- Create a time-limited download link for an object
- Check whether an object is publicly accessible and copy its public link
- Show object metadata (including user metadata) and object versions

## Usage

### Configuration

The application is configured with environment variables.

#### S3 instances

Every S3 account the app should manage is configured with a numbered set of
variables, starting at `S3_1_`. The app stops looking at the first number that
has no `NAME`, so the numbering must not have gaps. Each instance appears in
the app under its `NAME` and is reachable under `/<NAME>/buckets` (or
`/<NUMBER>/buckets`), so pick names that work in a URL:

```shell
S3_1_NAME=production
S3_1_ENDPOINT=s3.amazonaws.com
S3_1_ACCESS_KEY_ID=XXX
S3_1_SECRET_ACCESS_KEY=xxx

S3_2_NAME=backups
S3_2_ENDPOINT=minio.example.com:9000
S3_2_ACCESS_KEY_ID=YYY
S3_2_SECRET_ACCESS_KEY=yyy
```

A single instance may also be configured without a number (it is then named
`Default`), which is how earlier versions of the app were configured:

```shell
ENDPOINT=s3.amazonaws.com
ACCESS_KEY_ID=XXX
SECRET_ACCESS_KEY=xxx
```

The variables below are read per instance, either with an `S3_N_` prefix or,
for a single unnamed instance, without one. At least one instance must be
configured.

- `NAME`: The name the instance is shown and addressed under (required in the numbered form; a single unnamed instance is called `Default`)
- `ENDPOINT`: The endpoint of your S3 server (defaults to `s3.amazonaws.com`)
- `REGION`: The region of your S3 server (defaults to `""`)
- `ACCESS_KEY_ID`: Your S3 access key ID (required unless `USE_IAM` is `true` or `SIGNATURE_TYPE` is `Anonymous`)
- `SECRET_ACCESS_KEY`: Your S3 secret access key (required unless `USE_IAM` is `true` or `SIGNATURE_TYPE` is `Anonymous`)
- `USE_IAM`: Use IAM role instead of key pair (defaults to `false`)
- `IAM_ENDPOINT`: Endpoint for IAM role retrieving (Can be blank for AWS)
- `USE_SSL`: Whether your S3 server uses SSL or not (defaults to `true`)
- `SKIP_SSL_VERIFICATION`: Whether the HTTP client should skip SSL verification (defaults to `false`)
- `SIGNATURE_TYPE`: The signature type to be used (defaults to `V4`; valid values are `V2, V4, V4Streaming, Anonymous`). `Anonymous` sends unsigned requests, which is how public buckets are browsed
- `BUCKET_LOOKUP`: How buckets are addressed in requests (defaults to `Auto`; valid values are `Auto, DNS, Path`). `DNS` uses virtual-hosted–style addressing (`bucket.endpoint`), `Path` uses path-style addressing (`endpoint/bucket`) and `Auto` picks virtual-hosted style for Amazon and Google endpoints and path style for all others. Set it to `DNS` if your provider answers with `Virtual host domain is required while accessing a specific bucket`
- `PUBLIC_URL`: A template for the public links of objects, for when they are served from a CDN or a custom domain (defaults to unset, which links to the object on `ENDPOINT` following `BUCKET_LOOKUP`). `{key}` is replaced by the object key and `{bucket}` by the bucket name, for example `https://cdn.example.com/{key}` or `https://files.example.com/{bucket}/{key}`; `{key}` is required
- `BUCKETS`: A comma-separated list of bucket names to show in addition to the ones the instance lists itself (defaults to unset), for example public buckets owned by someone else. They are shown even if the instance refuses to list its buckets, as it does for anonymous access

#### Application

These variables apply to the whole app and are never prefixed:

- `PORT`: The port the app should listen on (defaults to `8080`)
- `ALLOW_DELETE`: Enable buttons to delete objects (defaults to `true`)
- `FORCE_DOWNLOAD`: Add response headers for object downloading instead of opening in a new tab (defaults to `true`; only affects the `Download` action, not `Open`)
- `LIST_RECURSIVE`: List all objects in buckets recursively (defaults to `false`)
- `SHOW_VERSIONS`: Show all object versions in bucket view and enable version-specific downloads (defaults to `false`; bucket must have versioning enabled)
- `SHOW_METADATA`: Show the object metadata action and enable the metadata endpoint (defaults to `true`)
- `TZ`: IANA timezone used when displaying object Last Modified times (defaults to UTC; for example `Europe/Berlin`)
- `BUCKET_NAME`: Restrict the buckets view to a single named bucket (defaults to unset, showing all buckets)
- `SSE_TYPE`: Specified server side encryption (defaults blank) Valid values can be `SSE`, `KMS`, `SSE-C` all others values don't enable the SSE
- `SSE_KEY`: The key needed for SSE method (only for `KMS` and `SSE-C`)
- `TIMEOUT`: The read and write timeout in seconds (default to `600` - 10 minutes)
- `ROOT_URL`: A root URL prefix if running behind a reverse proxy (defaults to unset)

### Browsing large buckets

The default bucket view asks S3 for one page of objects at a time, so opening a
bucket and stepping through it costs the same whether it holds ten objects or ten
million. Because S3 can only list keys in ascending order and offers no search of
its own, that page-at-a-time listing is possible only for the default view: it is
sorted by name ascending and unsearched. Such a view has no total object count
and no last page to jump to, since S3 never reports how much it did not return.

Sorting by another column, sorting descending, searching or choosing `All` items
per page needs the whole location in memory instead, and so does `SHOW_VERSIONS`.
Those listings stop after 10,000 objects and say so on the page — their counting,
sorting and searching then cover only that many. Narrow the listing down with a
search or by opening a folder to reach the rest.

### Browsing public buckets

Public buckets, such as the datasets of the
[Registry of Open Data on AWS](https://registry.opendata.aws/), can be browsed
without an account by configuring an instance for anonymous access and naming the
buckets to show:

```shell
SIGNATURE_TYPE=Anonymous
REGION=us-east-1
BUCKETS=noaa-ghcn-pds
```

Anonymous requests cannot look up the region of a bucket, so `REGION` has to
match it; a bucket in another region reports the region it is in. Any other
bucket can be opened by name with the folder button of the bucket list, even if
it isn't configured. Public buckets usually only allow reading, so uploading and
deleting fails on them.

### Build and Run Locally

1.  Run `make build`
1.  Execute the created binary and visit <http://localhost:8080>

### Run Container image

1. Run `docker run -p 8080:8080 -e 'ENDPOINT=s3.amazonaws.com' -e 'ACCESS_KEY_ID=XXX' -e 'SECRET_ACCESS_KEY=xxx' cloudlena/s3manager`

### Deploy to Kubernetes

You can deploy S3 Manager to a Kubernetes cluster using the [Helm chart](https://github.com/sergeyshevch/s3manager-helm).

#### Running behind a reverse proxy

If there are multiple S3 users/accounts in a site then multiple instances of the S3 manager can be run in Kubernetes and expose behind a single nginx reverse proxy ingress.
The s3manager can be run with a `ROOT_URL` environment variable set that accounts for the reverse proxy location.

If the nginx configuration block looks like:

```nginx
    location /teamx/ {
        proxy_pass http://s3manager-teamx:8080/;
        auth_basic "teamx";
        auth_basic_user_file /conf/teamx-htpasswd;
    }
    location /teamy/ {
        proxy_pass http://s3manager-teamy:8080/;
        <other nginx settings>
    }
```

Then the instance behind the `s3manager-teamx` service has `ROOT_URL=teamx` and the instance behind `s3manager-teamy` has `ROOT_URL=teamy`.
Other nginx settings can be applied to each location.
The nginx instance can be hosted on some reachable address and reverse proxy to the different S3 accounts.

## Development

### Lint Code

1. Run `make lint`

### Run Tests

1.  Run `make test`

### Build Container Image

The image is available on [Docker Hub](https://hub.docker.com/r/cloudlena/s3manager/).

1.  Run `make build-image`

### Run Locally for Testing

There is an example [docker-compose.yml](https://github.com/cloudlena/s3manager/blob/main/docker-compose.yml) file that spins up two S3 services and the S3 Manager configured for both of them. You can try it by issuing the following command:

```shell
$ docker-compose up
```

## GitHub Stars

[![GitHub stars over time](https://starchart.cc/cloudlena/s3manager.svg?variant=adaptive)](https://starchart.cc/cloudlena/s3manager)
