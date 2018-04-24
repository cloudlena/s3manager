FROM golang:latest

WORKDIR /go/src/github.com/mastertinner/s3manager
COPY . .

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
RUN go build -ldflags="-s -w" ./cmd/s3manager

EXPOSE 8080

ENTRYPOINT ["./s3manager"]
