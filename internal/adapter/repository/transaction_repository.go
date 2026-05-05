package repository

import (
	"context"
	"errors"

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
	var trx domain.Transaction
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&trx).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTransactionNotFound
		}
		return nil, err
	}
	return &trx, nil
}

func (r *transactionRepository) Save(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(t).Error
}

func (r *transactionRepository) Update(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Save(t).Error
}
