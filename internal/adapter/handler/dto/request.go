package dto

type ChargeRequestDTO struct {
	CustomerID      string `json:"customer_id" validate:"required,uuid"`
	PaymentMethodID string `json:"payment_method_id" validate:"required,uuid"`
	Amount          int64  `json:"amount" validate:"required,gt=0"`
	Currency        string `json:"currency" validate:"required,len=3"`
	IdempotencyKey  string `json:"-"` // Header
}

type WebhookRequestDTO struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	StatusCode        string `json:"status_code"`
	FraudStatus       string `json:"fraud_status"`
	TransactionStatus string `json:"transaction_status"`
	SignatureKey      string `json:"signature_key"`
}
