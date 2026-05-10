.PHONY: run build test test-base test-race migrate-up migrate-down docker-up docker-down

# Local execution
run:
	go run cmd/main.go

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service ./cmd/main.go

# Default test (menggunakan race detector)
test: test-race

# Test standar tanpa race detector (tidak butuh GCC/CGO)
test-base:
	go test -v ./...

# Test dengan race detector (memaksa CGO_ENABLED=1)
test-race:
	CGO_ENABLED=1 go test -v -race ./...

# Migrations
DB_URL="postgres://payment_user:payment_password@localhost:5432/payment_db?sslmode=disable"

migrate-up:
	migrate -path migrations -database $(DB_URL) up

migrate-down:
	migrate -path migrations -database $(DB_URL) down

# Docker compose wrappers
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down