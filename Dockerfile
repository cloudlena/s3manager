FROM golang:1 AS builder
RUN groupadd -r app && useradd --no-log-init -r -g app app
WORKDIR /app
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix cgo -o bin/s3manager ./cmd/s3manager

FROM scratch
WORKDIR /app
COPY --from=builder /app/bin/s3manager ./
COPY --from=builder /app/web ./web/
COPY --from=builder /etc/passwd /etc/passwd
USER app
EXPOSE 8080
ENTRYPOINT ["./s3manager"]
