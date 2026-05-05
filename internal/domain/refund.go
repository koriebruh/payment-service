package domain

import (
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
