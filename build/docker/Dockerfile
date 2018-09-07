FROM golang:1

WORKDIR /app
COPY . .

RUN go build -ldflags="-s -w" ./cmd/s3manager

EXPOSE 8080

ENTRYPOINT ["./s3manager"]
