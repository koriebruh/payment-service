// internal/usecase/charge_transaction_usecase.go
// Layer: Usecase
// Depends on: domain, usecase/port, usecase/dto, pkg/idempotency

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/idempotency"
)

const idempotencyTTL = 24 * time.Hour

type chargeTransactionUsecase struct {
	txManager        port.TxManager
	transactionRepo  port.TransactionRepository
	paymentGateway   port.PaymentGatewayPort
	idempotencyStore idempotency.IdempotencyStore
}

func NewChargeTransactionUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	paymentGateway port.PaymentGatewayPort,
	idempotencyStore idempotency.IdempotencyStore,
) port.ChargeTransactionUsecase {
	return &chargeTransactionUsecase{
		txManager:        txManager,
		transactionRepo:  transactionRepo,
		paymentGateway:   paymentGateway,
		idempotencyStore: idempotencyStore,
	}
}

func (u *chargeTransactionUsecase) Execute(ctx context.Context, req dto.ChargeRequest) (*dto.ChargeResult, error) {
	ctx, span := otel.Tracer("payment-service/usecase").Start(ctx, "chargeTransactionUsecase.Execute")
	defer span.End()

	// 1. Idempotency check — mandatory for all financial write operations (GEMINI.md rule 11)
	if u.idempotencyStore != nil && req.IdempotencyKey != "" {
		cached, err := u.idempotencyStore.Get(ctx, req.IdempotencyKey)
		if err == nil && cached != nil {
			var result dto.ChargeResult
			if jsonErr := json.Unmarshal(cached.Response, &result); jsonErr == nil {
				return &result, nil
			}
		}
	}

	now := time.Now()

	// 2. Build transaction with human-readable ID (INV/YYYYMMDD/TRX-XXXXXX)
	//    Order ID uses payment_method_id for traceability in logs
	trx := &domain.Transaction{
		ID:              domain.GenerateTransactionID(now),
		OrderID:         fmt.Sprintf("ORDER-%s-%d", req.PaymentMethodID, now.UnixNano()),
		CustomerID:      req.CustomerID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          domain.StatusPending,
		CorrelationID:   &req.TraceID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	customer := &domain.Customer{ID: req.CustomerID}

	// 3. Call Midtrans gateway
	updatedTrx, err := u.paymentGateway.CreateTransaction(ctx, trx, customer)
	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: create transaction at gateway: %w", err)
	}

	// 4. Persist — transaction boundary owned by usecase (GEMINI.md rule 7)
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

	// 5. Cache result for idempotency replay
	if u.idempotencyStore != nil && req.IdempotencyKey != "" {
		if resultBytes, jsonErr := json.Marshal(result); jsonErr == nil {
			if err := u.idempotencyStore.Set(ctx, req.IdempotencyKey, idempotency.IdempotencyRecord{
				Key:        req.IdempotencyKey,
				StatusCode: 201,
				Response:   resultBytes,
			}, idempotencyTTL); err != nil {
				slog.Warn("failed to set idempotency record", "key", req.IdempotencyKey, "error", err)
			}
		}
	}

	return result, nil
}
