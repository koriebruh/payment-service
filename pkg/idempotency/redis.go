package idempotency

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) IdempotencyStore {
	return &redisStore{
		client: client,
	}
}

func (s *redisStore) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	var record IdempotencyRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *redisStore) Set(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, ttl).Err()
}
