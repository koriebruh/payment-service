# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service ./cmd/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/payment-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./payment-service"]
