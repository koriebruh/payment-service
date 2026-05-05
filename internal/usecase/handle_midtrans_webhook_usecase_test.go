package usecase

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

// MockWebhookLogRepository
type MockWebhookLogRepository struct {
	mock.Mock
}

func (m *MockWebhookLogRepository) FindByMidtransTransactionIDAndStatus(ctx context.Context, trxID, status string) (*domain.WebhookLog, error) {
	args := m.Called(ctx, trxID, status)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.WebhookLog), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWebhookLogRepository) Save(ctx context.Context, tx port.Tx, log *domain.WebhookLog) error {
	args := m.Called(ctx, tx, log)
	return args.Error(0)
}

// MockOutboxRepository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Save(ctx context.Context, tx port.Tx, event *domain.OutboxEvent) error {
	args := m.Called(ctx, tx, event)
	return args.Error(0)
}

func (m *MockOutboxRepository) FindPendingEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*domain.OutboxEvent), args.Error(1)
}

func (m *MockOutboxRepository) MarkAsPublished(ctx context.Context, tx port.Tx, ids []string) error {
	args := m.Called(ctx, tx, ids)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkAsFailed(ctx context.Context, tx port.Tx, id string) error {
	args := m.Called(ctx, tx, id)
	return args.Error(0)
}

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	webhookLogRepo := new(MockWebhookLogRepository)
	outboxRepo := new(MockOutboxRepository)

	cfg := &config.Config{
		Midtrans: config.MidtransConfig{
			ServerKey: "secret-key",
		},
	}

	usecase := NewHandleMidtransWebhookUsecase(txManager, trxRepo, webhookLogRepo, outboxRepo, cfg)

	req := dto.WebhookPayload{
		OrderID:       "order-1",
		StatusCode:    "200",
		GrossAmount:   "10000.00",
		SignatureKey:  "invalid-signature",
	}

	txManager.On("WithTx", mock.Anything).Return(nil)
	webhookLogRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)

	err := usecase.Execute(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, domain.ErrInvalidSignature, err)
}

func TestHandleWebhook_SettlementSuccess(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	webhookLogRepo := new(MockWebhookLogRepository)
	outboxRepo := new(MockOutboxRepository)

	cfg := &config.Config{
		Midtrans: config.MidtransConfig{
			ServerKey: "secret-key",
		},
	}

	usecase := NewHandleMidtransWebhookUsecase(txManager, trxRepo, webhookLogRepo, outboxRepo, cfg)

	// Calculate valid signature
	payload := "order-1" + "200" + "10000.00" + "secret-key"
	hash := sha512.Sum512([]byte(payload))
	validSignature := hex.EncodeToString(hash[:])

	req := dto.WebhookPayload{
		TransactionID:     "midtrans-123",
		OrderID:           "order-1",
		StatusCode:        "200",
		GrossAmount:       "10000.00",
		SignatureKey:      validSignature,
		TransactionStatus: "settlement",
	}

	existingTrx := &domain.Transaction{
		ID:      "trx-1",
		OrderID: "order-1",
		Status:  domain.StatusPending,
	}

	txManager.On("WithTx", mock.Anything).Return(nil)
	webhookLogRepo.On("FindByMidtransTransactionIDAndStatus", mock.Anything, req.TransactionID, req.TransactionStatus).Return(nil, domain.NewNotFoundError("webhook_log", req.TransactionID))
	trxRepo.On("FindByOrderID", mock.Anything, req.OrderID).Return(existingTrx, nil)
	trxRepo.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil)
	webhookLogRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil)

	err := usecase.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, domain.StatusSettlement, existingTrx.Status)
}
