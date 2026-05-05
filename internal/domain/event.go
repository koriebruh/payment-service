package domain

import (
	"encoding/json"
	"time"
)

type EventEnvelope struct {
	EventID       string            `json:"event_id"`
	EventType     string            `json:"event_type"`
	SchemaVersion string            `json:"schema_version"`
	AggregateID   string            `json:"aggregate_id"`
	AggregateType string            `json:"aggregate_type"`
	ServiceSource string            `json:"service_source"`
	TraceID       string            `json:"trace_id"`
	CorrelationID string            `json:"correlation_id"`
	OccurredAt    time.Time         `json:"occurred_at"`
	PublishedAt   time.Time         `json:"published_at"`
	Payload       json.RawMessage   `json:"payload"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type PaymentSettledPayload struct {
	TransactionID string `json:"transaction_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	PaidAt        string `json:"paid_at"`
}
