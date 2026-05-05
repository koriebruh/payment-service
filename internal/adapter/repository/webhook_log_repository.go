package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type webhookLogRepository struct {
	db *gorm.DB
}

func NewWebhookLogRepository(db *gorm.DB) port.WebhookLogRepository {
	return &webhookLogRepository{db: db}
}

func (r *webhookLogRepository) FindByMidtransTransactionIDAndStatus(ctx context.Context, trxID, status string) (*domain.WebhookLog, error) {
	var log domain.WebhookLog
	err := r.db.WithContext(ctx).
		Where("midtrans_transaction_id = ? AND midtrans_status = ?", trxID, status).
		First(&log).Error
	
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("webhook_log", trxID)
		}
		return nil, err
	}
	return &log, nil
}

func (r *webhookLogRepository) Save(ctx context.Context, tx port.Tx, log *domain.WebhookLog) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(log).Error
}
