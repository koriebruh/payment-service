package domain

import (
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

func (t *Transaction) CanProcess() error {
	if t.Status != StatusPending {
		return ErrInvalidStatus
	}
	return nil
}
