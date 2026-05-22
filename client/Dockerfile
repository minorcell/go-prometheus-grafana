FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o client .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/client .
CMD ["./client", "-url", "http://server:8080", "-concurrency", "5", "-interval", "500ms"]
