package domain

import (
	"time"
)

type PaymentMethod struct {
	ID       string
	Name     string
	Type     string // ewallet, bank_transfer, card, qris
	IsActive bool
	Config   []byte // stored as JSONB
	CreatedAt time.Time
}
