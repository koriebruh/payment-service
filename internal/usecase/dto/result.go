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

type TransactionStatusResult struct {
	TransactionID   string
	OrderID         string
	Status          string
	Amount          int64
	Currency        string
	PaymentMethodID string
	PaidAt          *time.Time
	ExpiredAt       *time.Time
}

type RefundResult struct {
	RefundID      string
	TransactionID string
	Amount        int64
	Status        string
	Reason        *string
}

type PaymentMethodResult struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}
