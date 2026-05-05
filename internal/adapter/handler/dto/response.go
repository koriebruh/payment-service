package dto

type ChargeResponseDTO struct {
	TransactionID string  `json:"transaction_id"`
	OrderID       string  `json:"order_id"`
	Status        string  `json:"status"`
	PaymentURL    *string `json:"payment_url,omitempty"`
	SnapToken     *string `json:"snap_token,omitempty"`
}
