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
	"github.com/koriebruh/payment-service/pkg/idempotency"
)

// ─── Mock: TxManager ────────────────────────────────────────────────────────

type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(tx port.Tx) error) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		return fn(nil)
	}
	return args.Error(0)
}

// ─── Mock: TransactionRepository ────────────────────────────────────────────

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

// ─── Mock: PaymentGateway ────────────────────────────────────────────────────

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

// ─── Mock: PaymentMethodRepository ──────────────────────────────────────────

type MockPaymentMethodRepository struct {
	mock.Mock
}

func (m *MockPaymentMethodRepository) FindByID(ctx context.Context, id string) (*domain.PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.PaymentMethod), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPaymentMethodRepository) FindAllActive(ctx context.Context) ([]*domain.PaymentMethod, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.PaymentMethod), args.Error(1)
	}
	return nil, args.Error(1)
}

// ─── Mock: IdempotencyStore ──────────────────────────────────────────────────
// Matches the real idempotency.IdempotencyStore interface exactly.

type MockIdempotencyStore struct {
	mock.Mock
}

func (m *MockIdempotencyStore) Get(ctx context.Context, key string) (*idempotency.IdempotencyRecord, error) {
	args := m.Called(ctx, key)
	if args.Get(0) != nil {
		return args.Get(0).(*idempotency.IdempotencyRecord), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockIdempotencyStore) Set(ctx context.Context, key string, record idempotency.IdempotencyRecord, ttl time.Duration) error {
	args := m.Called(ctx, key, record, ttl)
	return args.Error(0)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildTestPaymentMethod(id string) *domain.PaymentMethod {
	return &domain.PaymentMethod{
		ID:     id,
		Name:   "QRIS",
		Type:   "E-WALLET",
		Config: []byte(`{"provider": "midtrans"}`),
	}
}

func buildTestChargeRequest() dto.ChargeRequest {
	return dto.ChargeRequest{
		CustomerID:      "customer-1",
		PaymentMethodID: "qris",
		Amount:          100000,
		Currency:        "IDR",
		IdempotencyKey:  "idem-key-001",
		TraceID:         "trace-001",
	}
}


// TestChargeTransactionUsecase_Success: happy path — full charge flow.
func TestChargeTransactionUsecase_Success(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	pmRepo := new(MockPaymentMethodRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	uc := NewChargeTransactionUsecase(txManager, trxRepo, pmRepo, gateway, idemStore)

	req := buildTestChargeRequest()

	snapToken := "snap-token-123"
	paymentURL := "https://app.sandbox.midtrans.com/snap/v4/redirection/abc"

	returnedTrx := &domain.Transaction{
		ID:              "trx-123",
		OrderID:         "ORDER-midtrans-qris-999",
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          domain.StatusPending,
		SnapToken:       &snapToken,
		PaymentURL:      &paymentURL,
		CreatedAt:       time.Now(),
	}

	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(nil, assert.AnError)
	pmRepo.On("FindByID", mock.Anything, req.PaymentMethodID).Return(buildTestPaymentMethod(req.PaymentMethodID), nil)
	gateway.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*domain.Transaction"), mock.AnythingOfType("*domain.Customer")).Return(returnedTrx, nil)
	txManager.On("WithTx", mock.Anything).Return(nil)
	trxRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil)
	idemStore.On("Set", mock.Anything, req.IdempotencyKey, mock.AnythingOfType("idempotency.IdempotencyRecord"), idempotencyTTL).Return(nil)

	res, err := uc.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "trx-123", res.TransactionID)
	assert.Equal(t, string(domain.StatusPending), res.Status)
	assert.NotNil(t, res.SnapToken)
	assert.NotNil(t, res.PaymentURL)

	idemStore.AssertExpectations(t)
	pmRepo.AssertExpectations(t)
	gateway.AssertExpectations(t)
	txManager.AssertExpectations(t)
	trxRepo.AssertExpectations(t)
}

// TestChargeTransactionUsecase_IdempotencyHit: duplicate request returns cached result immediately.
func TestChargeTransactionUsecase_IdempotencyHit(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	pmRepo := new(MockPaymentMethodRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	uc := NewChargeTransactionUsecase(txManager, trxRepo, pmRepo, gateway, idemStore)

	req := buildTestChargeRequest()

	// Simulate a previously cached result
	cachedPayload := []byte(`{"transaction_id":"trx-cached","order_id":"ORDER-midtrans-qris-111","status":"pending","created_at":"2026-01-01T00:00:00Z"}`)
	cached := &idempotency.IdempotencyRecord{
		Key:        req.IdempotencyKey,
		StatusCode: 201,
		Response:   cachedPayload,
	}

	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(cached, nil)

	res, err := uc.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "trx-cached", res.TransactionID)

	// Gateway and DB must NOT be called on idempotency hit
	gateway.AssertNotCalled(t, "CreateTransaction")
	txManager.AssertNotCalled(t, "WithTx")
	trxRepo.AssertNotCalled(t, "Save")
}

// TestChargeTransactionUsecase_InvalidPaymentMethod: returns domain error when pm not found.
func TestChargeTransactionUsecase_InvalidPaymentMethod(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	pmRepo := new(MockPaymentMethodRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	uc := NewChargeTransactionUsecase(txManager, trxRepo, pmRepo, gateway, idemStore)

	req := buildTestChargeRequest()
	req.PaymentMethodID = "unknown-method"

	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(nil, assert.AnError)
	pmRepo.On("FindByID", mock.Anything, req.PaymentMethodID).Return(nil, assert.AnError)

	res, err := uc.Execute(context.Background(), req)

	assert.Nil(t, res)
	assert.Error(t, err)
	gateway.AssertNotCalled(t, "CreateTransaction")
}
