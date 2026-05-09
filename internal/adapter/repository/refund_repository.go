package repository

import (
	"context"

	"go.opentelemetry.io/otel"
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
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "refundRepository.Save")
	defer span.End()

	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(refund).Error
}

func (r *refundRepository) Update(ctx context.Context, tx port.Tx, refund *domain.Refund) error {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "refundRepository.Update")
	defer span.End()

	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Save(refund).Error
}

func (r *refundRepository) FindByTransactionID(ctx context.Context, transactionID string) ([]*domain.Refund, error) {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "refundRepository.FindByTransactionID")
	defer span.End()

	var refunds []*domain.Refund
	err := r.db.WithContext(ctx).Where("transaction_id = ?", transactionID).Order("created_at DESC").Find(&refunds).Error
	return refunds, err
}
