FROM golang:1.17 as builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o app .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/app/app .
VOLUME /data

ENTRYPOINT ["./app"]
