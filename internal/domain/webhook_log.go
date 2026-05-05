package domain

import (
	"time"
)

type WebhookLog struct {
	ID                    string
	TransactionID         *string
	EventType             *string
	MidtransTransactionID *string
	MidtransStatus        *string
	FraudStatus           *string
	RawPayload            string
	SignatureValid        string // "valid", "invalid", "unknown"
	ReceivedAt            time.Time
}
