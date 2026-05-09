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

// ─── Mock: TxManager ─────────────────────────────────────────────────────────

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

// ─── Mock: TransactionRepository ─────────────────────────────────────────────

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

func (m *MockTransactionRepository) FindByOrderIDForUpdate(ctx context.Context, tx port.Tx, orderID string) (*domain.Transaction, error) {
	args := m.Called(ctx, tx, orderID)
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

// ─── Mock: PaymentGateway ─────────────────────────────────────────────────────

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

// ─── Mock: IdempotencyStore ───────────────────────────────────────────────────
// Matches idempotency.IdempotencyStore interface exactly.

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

// ─── Helpers ──────────────────────────────────────────────────────────────────

func buildTestChargeRequest() dto.ChargeRequest {
	return dto.ChargeRequest{
		CustomerID:      "550e8400-e29b-41d4-a716-446655440001",
		PaymentMethodID: "qris",
		Amount:          100000,
		Currency:        "IDR",
		IdempotencyKey:  "idem-key-001",
		TraceID:         "trace-001",
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestChargeTransactionUsecase_Success: happy path — full charge flow with Midtrans.
// Verifies that:
//   - transaction ID follows INV/YYYYMMDD/TRX-XXXXXX pattern
//   - gateway is called once
//   - transaction is persisted
//   - idempotency result is cached
func TestChargeTransactionUsecase_Success(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	// 4 args — pmRepo removed since multi-gateway was cancelled
	uc := NewChargeTransactionUsecase(txManager, trxRepo, gateway, idemStore)

	req := buildTestChargeRequest()

	snapToken := "snap-token-123"
	paymentURL := "https://app.sandbox.midtrans.com/snap/v4/redirection/abc"

	// The gateway returns the transaction with its (gateway-enriched) fields.
	// We use AnythingOfType to match the trx built inside the usecase.
	returnedTrx := &domain.Transaction{
		ID:              domain.GenerateTransactionID(time.Now()),
		OrderID:         "ORDER-qris-1777991352878176829",
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          domain.StatusPending,
		SnapToken:       &snapToken,
		PaymentURL:      &paymentURL,
		CreatedAt:       time.Now(),
	}

	// Setup mock expectations in execution order
	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(nil, assert.AnError)
	gateway.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*domain.Transaction"), mock.AnythingOfType("*domain.Customer")).Return(returnedTrx, nil)
	txManager.On("WithTx", mock.Anything).Return(nil)
	trxRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*domain.Transaction")).Return(nil)
	idemStore.On("Set", mock.Anything, req.IdempotencyKey, mock.AnythingOfType("idempotency.IdempotencyRecord"), idempotencyTTL).Return(nil)

	res, err := uc.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Verify ID follows INV/YYYYMMDD/TRX-XXXXXX pattern
	assert.Regexp(t, `^INV/\d{8}/TRX-\d{6}$`, res.TransactionID,
		"transaction ID must match INV/YYYYMMDD/TRX-XXXXXX pattern")
	assert.Equal(t, string(domain.StatusPending), res.Status)
	assert.NotNil(t, res.SnapToken)
	assert.NotNil(t, res.PaymentURL)

	idemStore.AssertExpectations(t)
	gateway.AssertExpectations(t)
	txManager.AssertExpectations(t)
	trxRepo.AssertExpectations(t)
}

// TestChargeTransactionUsecase_IdempotencyHit: duplicate request returns cached result.
// Verifies that gateway and DB are NOT called on a cache hit.
func TestChargeTransactionUsecase_IdempotencyHit(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	uc := NewChargeTransactionUsecase(txManager, trxRepo, gateway, idemStore)

	req := buildTestChargeRequest()

	// Simulate a previously cached result with the new ID pattern
	cachedPayload := []byte(`{
		"transaction_id":"INV/20260506/TRX-004821",
		"order_id":"ORDER-qris-111",
		"status":"pending",
		"created_at":"2026-01-01T00:00:00Z"
	}`)
	cached := &idempotency.IdempotencyRecord{
		Key:        req.IdempotencyKey,
		StatusCode: 201,
		Response:   cachedPayload,
	}

	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(cached, nil)

	res, err := uc.Execute(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "INV/20260506/TRX-004821", res.TransactionID)

	// Gateway and DB must NOT be called on idempotency hit
	gateway.AssertNotCalled(t, "CreateTransaction")
	txManager.AssertNotCalled(t, "WithTx")
	trxRepo.AssertNotCalled(t, "Save")
}

// TestChargeTransactionUsecase_GatewayError: gateway failure returns wrapped error.
func TestChargeTransactionUsecase_GatewayError(t *testing.T) {
	txManager := new(MockTxManager)
	trxRepo := new(MockTransactionRepository)
	gateway := new(MockPaymentGateway)
	idemStore := new(MockIdempotencyStore)

	uc := NewChargeTransactionUsecase(txManager, trxRepo, gateway, idemStore)

	req := buildTestChargeRequest()

	idemStore.On("Get", mock.Anything, req.IdempotencyKey).Return(nil, assert.AnError)
	gateway.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*domain.Transaction"), mock.AnythingOfType("*domain.Customer")).
		Return(nil, assert.AnError)

	res, err := uc.Execute(context.Background(), req)

	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create transaction at gateway")

	// DB must NOT be called if gateway fails
	txManager.AssertNotCalled(t, "WithTx")
	trxRepo.AssertNotCalled(t, "Save")
}

// TestGenerateTransactionID: domain factory produces correct pattern.
func TestGenerateTransactionID(t *testing.T) {
	now := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	id := domain.GenerateTransactionID(now)
	assert.Regexp(t, `^INV/20260506/TRX-\d{6}$`, id)
}

// TestGenerateRefundID: domain factory produces correct pattern.
func TestGenerateRefundID(t *testing.T) {
	now := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	id := domain.GenerateRefundID(now)
	assert.Regexp(t, `^REF/20260506/\d{4}$`, id)
}
