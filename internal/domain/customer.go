package domain

import (
	"time"
)

type Customer struct {
	ID        string
	Name      string
	Email     string
	Phone     string
	CreatedAt time.Time
}
