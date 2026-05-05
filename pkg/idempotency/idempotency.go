package idempotency

import (
	"context"
	"time"
)

type IdempotencyRecord struct {
	Key        string `json:"key"`
	StatusCode int    `json:"status_code"`
	Response   []byte `json:"response"`
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)
	Set(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error
}
