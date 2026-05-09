package repository

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) port.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.FindByID")
	defer span.End()

	var trx domain.Transaction
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&trx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return &trx, nil
}

func (r *transactionRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Transaction, error) {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.FindByOrderID")
	defer span.End()

	var t domain.Transaction
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&t).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *transactionRepository) FindByOrderIDForUpdate(ctx context.Context, tx port.Tx, orderID string) (*domain.Transaction, error) {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.FindByOrderIDForUpdate")
	defer span.End()

	db := GetGormDB(r.db, tx)
	var t domain.Transaction
	// SELECT ... FOR UPDATE — pessimistic lock to prevent concurrent webhook processing
	err := db.WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE").
		Where("order_id = ?", orderID).
		First(&t).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *transactionRepository) FindAll(ctx context.Context, customerID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.FindAll")
	defer span.End()

	var transactions []*domain.Transaction
	var totalCount int64

	query := r.db.WithContext(ctx).Model(&domain.Transaction{})
	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error
	return transactions, totalCount, err
}

func (r *transactionRepository) Save(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.Save")
	defer span.End()

	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(t).Error
}

func (r *transactionRepository) Update(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	ctx, span := otel.Tracer("payment-service/repository").Start(ctx, "transactionRepository.Update")
	defer span.End()

	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Save(t).Error
}
