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
	FindAll(ctx context.Context, customerID string, limit, offset int) ([]*domain.Transaction, int64, error)
	Save(ctx context.Context, tx Tx, t *domain.Transaction) error
	Update(ctx context.Context, tx Tx, t *domain.Transaction) error
}

type WebhookLogRepository interface {
	FindByMidtransTransactionIDAndStatus(ctx context.Context, trxID, status string) (*domain.WebhookLog, error)
	Save(ctx context.Context, tx Tx, log *domain.WebhookLog) error
}

type PaymentGatewayPort interface {
	CreateTransaction(ctx context.Context, t *domain.Transaction, c *domain.Customer) (*domain.Transaction, error)
	RefundTransaction(ctx context.Context, orderID string, refund *domain.Refund) (*domain.Refund, error)
	CancelTransaction(ctx context.Context, orderID string) error
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

type PaymentMethodRepository interface {
	FindByID(ctx context.Context, id string) (*domain.PaymentMethod, error)
	FindAllActive(ctx context.Context) ([]*domain.PaymentMethod, error)
}

type RefundRepository interface {
	Save(ctx context.Context, tx Tx, r *domain.Refund) error
	Update(ctx context.Context, tx Tx, r *domain.Refund) error
	FindByTransactionID(ctx context.Context, transactionID string) ([]*domain.Refund, error)
}
