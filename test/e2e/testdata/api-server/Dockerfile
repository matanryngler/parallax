FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY main.go .
RUN go mod init api-server && \
    go build -o api-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api-server .
EXPOSE 8080
CMD ["./api-server"]