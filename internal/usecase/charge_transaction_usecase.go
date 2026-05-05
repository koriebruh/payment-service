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
	pm, err := u.pmRepo.FindByID(ctx, req.PaymentMethodID)
	if err != nil {
		return nil, domain.NewValidationError(fmt.Sprintf("invalid payment method: %s", req.PaymentMethodID))
	}

	// Extract provider from JSONB config. Default to midtrans if not found.
	provider := "midtrans" // fallback
	if len(pm.Config) > 0 {
		var configMap map[string]interface{}
		if err := json.Unmarshal(pm.Config, &configMap); err == nil {
			if providerVal, ok := configMap["provider"]; ok {
				if pStr, ok := providerVal.(string); ok {
					provider = pStr
				}
			}
		}
	}

	// Create a clear order ID: ORDER-{provider}-{method}-{timestamp}
	orderID := fmt.Sprintf("ORDER-%s-%s-%d", provider, req.PaymentMethodID, time.Now().UnixNano())

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

	customer := &domain.Customer{
		ID: req.CustomerID,
		// In a real app we might fetch customer details here
	}

	// Midtrans call
	updatedTrx, err := u.paymentGateway.CreateTransaction(ctx, trx, customer)
	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: create transaction at gateway: %w", err)
	}

	err = u.txManager.WithTx(ctx, func(tx port.Tx) error {
		if err := u.transactionRepo.Save(ctx, tx, updatedTrx); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("chargeTransactionUsecase.Execute: save transaction: %w", err)
	}

	return &dto.ChargeResult{
		TransactionID: updatedTrx.ID,
		OrderID:       updatedTrx.OrderID,
		Status:        string(updatedTrx.Status),
		PaymentURL:    updatedTrx.PaymentURL,
		SnapToken:     updatedTrx.SnapToken,
		CreatedAt:     updatedTrx.CreatedAt,
	}, nil
}
