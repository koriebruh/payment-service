package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type outboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) port.OutboxRepository {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) Save(ctx context.Context, tx port.Tx, event *domain.OutboxEvent) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).Create(event).Error
}

func (r *outboxRepository) FindPendingEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	var events []*domain.OutboxEvent
	// Order by created_at to process oldest first
	err := r.db.WithContext(ctx).
		Where("status = ?", domain.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *outboxRepository) MarkAsPublished(ctx context.Context, tx port.Tx, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     domain.OutboxStatusPublished,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *outboxRepository) MarkAsFailed(ctx context.Context, tx port.Tx, id string) error {
	db := GetGormDB(r.db, tx)
	return db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     domain.OutboxStatusFailed,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}
