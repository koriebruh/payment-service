package domain

import (
	"encoding/json"
	"time"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage // JSONB in DB
	Status        OutboxStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
