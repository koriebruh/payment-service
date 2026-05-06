package domain

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type RefundStatus string

const (
	RefundStatusPending RefundStatus = "pending"
	RefundStatusSuccess RefundStatus = "success"
	RefundStatusFailed  RefundStatus = "failed"
)

type Refund struct {
	ID                string
	TransactionID     string
	Amount            int64
	Reason            *string
	Status            RefundStatus
	MidtransRefundKey *string
	MidtransResponse  []byte // JSONB
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// GenerateRefundID produces a human-readable refund ID.
// Pattern: REF/{YYYYMMDD}/{4-digit-random}
// Example: REF/20260506/4821
func GenerateRefundID(t time.Time) string {
	seq := rand.IntN(9000) + 1000 // always 4 digits: 1000–9999
	return fmt.Sprintf("REF/%s/%04d", t.Format("20060102"), seq)
}
