FROM golang:1.25-rc-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download && go build -o /app/redis-rate-limiter

RUN chmod +x /app/redis-rate-limiter

FROM alpine:latest

COPY --from=builder /app/redis-rate-limiter /usr/local/bin/redis-rate-limiter

ENTRYPOINT ["/usr/local/bin/redis-rate-limiter"]