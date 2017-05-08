FROM golang

ADD . /go/src/github.com/mastertinner/s3manager
WORKDIR /go/src/github.com/mastertinner/s3manager

RUN go build ./cmd/s3manager

EXPOSE 8080

CMD ./s3manager -endpoint "${S3_ENDPOINT}" -access-key-id "${S3_ACCESS_KEY_ID}" -secret-access-key "${S3_SECRET_ACCESS_KEY}"
