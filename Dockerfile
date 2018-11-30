FROM golang:1 as builder
RUN groupadd -r s3manager && useradd --no-log-init -r -g s3manager s3manager
WORKDIR /usr/src/s3manager
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -a -installsuffix cgo -o bin/s3manager ./cmd/s3manager

FROM scratch
WORKDIR /usr/s3manager
COPY --from=builder /usr/src/s3manager/bin/s3manager /usr/src/s3manager/web ./
COPY --from=builder /etc/passwd /etc/passwd
USER s3manager
EXPOSE 8080
ENTRYPOINT ["./s3manager"]
