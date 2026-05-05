package domain

import (
	"time"
)

type PaymentMethod struct {
	ID       string
	Code     string // gopay, qris, bca_va
	Name     string
	Type     string // ewallet, bank_transfer, card, qris
	IsActive bool
	Config   []byte // stored as JSONB
	CreatedAt time.Time
}
