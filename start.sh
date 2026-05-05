#!/bin/sh

set -e

echo "Running database migrations..."
migrate -path /app/migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up

echo "Starting payment-service..."
exec "$@"
