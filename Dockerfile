# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service ./cmd/main.go

# Install golang-migrate
RUN apk add --no-cache curl \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.15.2/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate /usr/local/bin/migrate

# Run stage
FROM alpine:latest

# Install make and postgresql-client
RUN apk add --no-cache make postgresql-client

WORKDIR /app

# Copy binary and migrations
COPY --from=builder /app/payment-service .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /usr/local/bin/migrate /usr/local/bin/migrate

# Copy start script
COPY start.sh .
RUN chmod +x start.sh

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
CMD ["./payment-service"]
