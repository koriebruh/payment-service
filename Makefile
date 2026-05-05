.PHONY: run build test migrate-up migrate-down docker-up docker-down

# Local execution
run:
	go run cmd/main.go

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service ./cmd/main.go

test:
	go test -v ./...

# Migrations (requires golang-migrate CLI installed locally)
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
