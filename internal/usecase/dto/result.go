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

type RefundDetail struct {
	RefundID  string     `json:"refund_id"`
	Amount    int64      `json:"amount"`
	Reason    *string    `json:"reason,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

type TransactionStatusResult struct {
	TransactionID   string                 `json:"transaction_id"`
	OrderID         string                 `json:"order_id"`
	Status          string                 `json:"status"`
	Amount          int64                  `json:"amount"`
	Currency        string                 `json:"currency"`
	PaymentMethodID string                 `json:"payment_method_id"`
	PaidAt          *time.Time             `json:"paid_at,omitempty"`
	ExpiredAt       *time.Time             `json:"expired_at,omitempty"`
	PaymentDetails  map[string]interface{} `json:"payment_details,omitempty"`
	Refunds         []RefundDetail         `json:"refunds"`
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
	Name string `json:"name"`
	Type string `json:"type"`
}

type TransactionListResult struct {
	Data       []*TransactionStatusResult `json:"data"`
	TotalCount int64                      `json:"total_count"`
	Limit      int                        `json:"limit"`
	Offset     int                        `json:"offset"`
}
