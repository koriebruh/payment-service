package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type refundRepository struct {
	db *gorm.DB
}

func NewRefundRepository(db *gorm.DB) port.RefundRepository {
	return &refundRepository{db: db}
}

func (r *refundRepository) Save(ctx context.Context, tx port.Tx, refund *domain.Refund) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(refund).Error
}

func (r *refundRepository) Update(ctx context.Context, tx port.Tx, refund *domain.Refund) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Save(refund).Error
}
