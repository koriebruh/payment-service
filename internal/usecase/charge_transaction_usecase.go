// internal/usecase/charge_transaction_usecase.go
// Layer: Usecase
// Depends on: domain, usecase/port, usecase/dto, pkg/idempotency

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/idempotency"
)

const (
	idempotencyTTL     = 24 * time.Hour
	defaultGatewayName = "midtrans"
)

type chargeTransactionUsecase struct {
	txManager        port.TxManager
	transactionRepo  port.TransactionRepository
	pmRepo           port.PaymentMethodRepository
	paymentGateway   port.PaymentGatewayPort
	idempotencyStore idempotency.IdempotencyStore
}

func NewChargeTransactionUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	pmRepo port.PaymentMethodRepository,
	paymentGateway port.PaymentGatewayPort,
	idempotencyStore idempotency.IdempotencyStore,
) port.ChargeTransactionUsecase {
	return &chargeTransactionUsecase{
		txManager:        txManager,
		transactionRepo:  transactionRepo,
		pmRepo:           pmRepo,
		paymentGateway:   paymentGateway,
		idempotencyStore: idempotencyStore,
	}
}

func (u *chargeTransactionUsecase) Execute(ctx context.Context, req dto.ChargeRequest) (*dto.ChargeResult, error) {
	// 1. Idempotency check (GEMINI.md rule 11 — mandatory for all write operations)
	if u.idempotencyStore != nil && req.IdempotencyKey != "" {
		cached, err := u.idempotencyStore.Get(ctx, req.IdempotencyKey)
		if err == nil && cached != nil {
			var result dto.ChargeResult
			if jsonErr := json.Unmarshal(cached.Response, &result); jsonErr == nil {
				return &result, nil
			}
		}
	}

	// 2. Validate payment method exists and extract provider for order ID prefix
	pm, err := u.pmRepo.FindByID(ctx, req.PaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: find payment method: %w",
			domain.NewValidationError(fmt.Sprintf("invalid payment method: %s", req.PaymentMethodID)))
	}

	// 3. Resolve gateway name from JSONB config — supports multi-gateway routing in the future.
	//    Currently only Midtrans is wired. Add new adapters by updating PaymentGatewayPort.
	gatewayName := extractProvider(pm.Config)

	// 4. Build order ID: ORDER-{gateway}-{method}-{nanoseconds}
	orderID := fmt.Sprintf("ORDER-%s-%s-%d", gatewayName, req.PaymentMethodID, time.Now().UnixNano())

	trx := &domain.Transaction{
		ID:              uuid.NewString(),
		OrderID:         orderID,
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          domain.StatusPending,
		CorrelationID:   &req.TraceID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	customer := &domain.Customer{ID: req.CustomerID}

	// 5. Call payment gateway (currently Midtrans; gateway is injected via PaymentGatewayPort)
	updatedTrx, err := u.paymentGateway.CreateTransaction(ctx, trx, customer)
	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: create transaction at gateway: %w", err)
	}

	// 6. Persist — transaction owned by usecase (GEMINI.md rule 7)
	err = u.txManager.WithTx(ctx, func(tx port.Tx) error {
		if err := u.transactionRepo.Save(ctx, tx, updatedTrx); err != nil {
			return fmt.Errorf("save transaction: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: persist transaction: %w", err)
	}

	result := &dto.ChargeResult{
		TransactionID: updatedTrx.ID,
		OrderID:       updatedTrx.OrderID,
		Status:        string(updatedTrx.Status),
		PaymentURL:    updatedTrx.PaymentURL,
		SnapToken:     updatedTrx.SnapToken,
		CreatedAt:     updatedTrx.CreatedAt,
	}

	// 7. Cache result for idempotency
	if u.idempotencyStore != nil && req.IdempotencyKey != "" {
		if resultBytes, jsonErr := json.Marshal(result); jsonErr == nil {
			_ = u.idempotencyStore.Set(ctx, req.IdempotencyKey, idempotency.IdempotencyRecord{
				Key:        req.IdempotencyKey,
				StatusCode: 201,
				Response:   resultBytes,
			}, idempotencyTTL)
		}
	}

	return result, nil
}

// extractProvider reads the "provider" field from a JSONB payment method config.
// Defaults to "midtrans" if absent or unparseable.
// This is the hook point for multi-gateway routing without touching usecase logic.
func extractProvider(config []byte) string {
	if len(config) == 0 {
		return defaultGatewayName
	}
	var cfg map[string]any
	if err := json.Unmarshal(config, &cfg); err != nil {
		return defaultGatewayName
	}
	if val, ok := cfg["provider"]; ok {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return defaultGatewayName
}
