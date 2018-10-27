FROM golang:1 as builder
WORKDIR /usr/src/app
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix cgo -o bin/s3manager ./cmd/s3manager

FROM scratch
WORKDIR /usr/app
COPY --from=builder /usr/src/app/bin/s3manager ./
COPY --from=builder /usr/src/app/web ./
EXPOSE 8080
ENTRYPOINT ["./s3manager"]

