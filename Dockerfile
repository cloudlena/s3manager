FROM docker.io/library/golang:1.26.4-alpine AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -a \
    -installsuffix cgo \
    -o bin/s3manager

FROM docker.io/library/alpine:3.23

WORKDIR /usr/src/app

RUN apk update \
    && apk upgrade --no-cache \
    && apk add --no-cache \
        ca-certificates \
        dumb-init \
    && addgroup -S s3manager \
    && adduser -S s3manager -G s3manager \
    && rm -rf /var/cache/apk/*

COPY --from=builder --chown=s3manager:s3manager /usr/src/app/bin/s3manager ./

USER s3manager

EXPOSE 8080

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["/usr/src/app/s3manager"]