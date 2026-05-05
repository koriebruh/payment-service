package port

import (
	"context"

	"github.com/koriebruh/payment-service/internal/domain"
)

type Tx interface {
	Commit() error
	Rollback() error
}

type TxManager interface {
	WithTx(ctx context.Context, fn func(tx Tx) error) error
}

type TransactionRepository interface {
	FindByID(ctx context.Context, id string) (*domain.Transaction, error)
	FindByOrderID(ctx context.Context, orderID string) (*domain.Transaction, error)
	Save(ctx context.Context, tx Tx, t *domain.Transaction) error
	Update(ctx context.Context, tx Tx, t *domain.Transaction) error
}

type WebhookLogRepository interface {
	FindByMidtransTransactionIDAndStatus(ctx context.Context, trxID, status string) (*domain.WebhookLog, error)
	Save(ctx context.Context, tx Tx, log *domain.WebhookLog) error
}

type PaymentGatewayPort interface {
	CreateTransaction(ctx context.Context, t *domain.Transaction, c *domain.Customer) (*domain.Transaction, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, event domain.EventEnvelope) error
}

type OutboxRepository interface {
	Save(ctx context.Context, tx Tx, event *domain.OutboxEvent) error
	FindPendingEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkAsPublished(ctx context.Context, tx Tx, ids []string) error
	MarkAsFailed(ctx context.Context, tx Tx, id string) error
}
