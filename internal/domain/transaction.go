package domain

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusSettlement TransactionStatus = "settlement"
	StatusCancel     TransactionStatus = "cancel"
	StatusDeny       TransactionStatus = "deny"
	StatusExpire     TransactionStatus = "expire"
	StatusFailure    TransactionStatus = "failure"
)

type Transaction struct {
	ID                    string
	OrderID               string
	CustomerID            string
	PaymentMethodID       string
	Amount                int64
	Currency              string
	Status                TransactionStatus
	MidtransTransactionID *string
	MidtransOrderID       *string
	PaymentURL            *string
	SnapToken             *string
	PaidAt                *time.Time
	ExpiredAt             *time.Time
	MidtransResponse      []byte // JSONB
	CorrelationID         *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// GenerateTransactionID produces a human-readable transaction ID.
// Pattern: INV/{YYYYMMDD}/TRX-{6-digit-random}
// Example: INV/20260506/TRX-004821
func GenerateTransactionID(t time.Time) string {
	seq := rand.IntN(900000) + 100000 // always 6 digits: 100000–999999
	return fmt.Sprintf("INV/%s/TRX-%06d", t.Format("20060102"), seq)
}

func (t *Transaction) CanProcess() error {
	if t.Status != StatusPending {
		return ErrInvalidStatus
	}
	return nil
}
