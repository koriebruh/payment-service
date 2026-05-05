package idempotency

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRecordNotFound = errors.New("idempotency record not found")

type memoryStore struct {
	mu      sync.RWMutex
	records map[string]IdempotencyRecord
}

func NewMemoryStore() IdempotencyStore {
	return &memoryStore{
		records: make(map[string]IdempotencyRecord),
	}
}

func (s *memoryStore) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if record, exists := s.records[key]; exists {
		return &record, nil
	}
	return nil, ErrRecordNotFound
}

func (s *memoryStore) Set(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[key] = record
	return nil
}
