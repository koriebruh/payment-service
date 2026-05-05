package dto

import "time"

type ChargeResult struct {
	TransactionID string
	OrderID       string
	Status        string
	PaymentURL    *string
	SnapToken     *string
	CreatedAt     time.Time
}
