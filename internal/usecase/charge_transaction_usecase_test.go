package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

// MockTxManager
type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(tx port.Tx) error) error {
	args := m.Called(ctx)
	// Execute the function with a dummy transaction if no error is set to fail
	if args.Error(0) == nil {
		return fn(nil)
	}
	return args.Error(0)
}

// MockTransactionRepository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTransactionRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Transaction, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockTransactionRepository) FindAll(ctx context.Context, customerID string, limit, offset int) ([]*domain.Transaction, int64, error) {
	args := m.Called(ctx, customerID, limit, offset)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.Transaction), args.Get(1).(int64), args.Error(2)
	}
	return nil, 0, args.Error(2)
}

func (m *MockTransactionRepository) Save(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	args := m.Called(ctx, tx, t)
	return args.Error(0)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx port.Tx, t *domain.Transaction) error {
	args := m.Called(ctx, tx, t)
	return args.Error(0)
}

// MockPaymentGateway
type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) CreateTransaction(ctx context.Context, t *domain.Transaction, c *domain.Customer) (*domain.Transaction, error) {
	args := m.Called(ctx, t, c)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPaymentGateway) RefundTransaction(ctx context.Context, orderID string, refund *domain.Refund) (*domain.Refund, error) {
	args := m.Called(ctx, orderID, refund)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Refund), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPaymentGateway) CancelTransaction(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func TestChargeTransactionUsecase_Success(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	gateway := new(MockPaymentGateway)
	
	usecase := NewChargeTransactionUsecase(txManager, trxRepo, gateway, nil) // passing nil for idempotency for now
	
	req := dto.ChargeRequest{
		CustomerID:      "customer-1",
		PaymentMethodID: "pm-1",
		Amount:          100000,
		Currency:        "IDR",
		IdempotencyKey:  "idem-1",
	}

	snapToken := "snap-token-123"
	paymentURL := "https://midtrans.com/pay"

	expectedTrx := &domain.Transaction{
		ID:              "trx-123",
		OrderID:         "ORDER-cust-1-123",
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          domain.StatusPending,
		SnapToken:       &snapToken,
		PaymentURL:      &paymentURL,
		CreatedAt:       time.Now(),
	}

	gateway.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*domain.Transaction"), mock.AnythingOfType("*domain.Customer")).Return(expectedTrx, nil)
	txManager.On("WithTx", mock.Anything).Return(nil)
	trxRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil)

	res, err := usecase.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "trx-123", res.TransactionID)
	assert.Equal(t, domain.StatusPending, domain.TransactionStatus(res.Status))
	
	gateway.AssertExpectations(t)
	txManager.AssertExpectations(t)
	trxRepo.AssertExpectations(t)
}
