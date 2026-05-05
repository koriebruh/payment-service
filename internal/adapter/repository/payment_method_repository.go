package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type paymentMethodRepository struct {
	db *gorm.DB
}

func NewPaymentMethodRepository(db *gorm.DB) port.PaymentMethodRepository {
	return &paymentMethodRepository{db: db}
}

func (r *paymentMethodRepository) FindAllActive(ctx context.Context) ([]*domain.PaymentMethod, error) {
	var methods []*domain.PaymentMethod
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&methods).Error
	return methods, err
}
