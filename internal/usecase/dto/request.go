package dto

type ChargeRequest struct {
	CustomerID      string
	PaymentMethodID string
	Amount          int64
	Currency        string
	IdempotencyKey  string
	RequestID       string
	TraceID         string
}

type WebhookPayload struct {
	TransactionID string
	OrderID       string
	GrossAmount   string
	StatusCode    string
	FraudStatus   string
	TransactionStatus string
	SignatureKey  string
	RawPayload    string
}

type RefundRequest struct {
	OrderID        string
	Amount         int64
	Reason         string
	IdempotencyKey string
}

type ListTransactionsRequest struct {
	CustomerID string
	Limit      int
	Offset     int
}
