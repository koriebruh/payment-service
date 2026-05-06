package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/idempotency"
)

type refundTransactionUsecase struct {
	txManager        port.TxManager
	transactionRepo  port.TransactionRepository
	refundRepo       port.RefundRepository
	paymentGateway   port.PaymentGatewayPort
	idempotencyStore idempotency.IdempotencyStore
}

func NewRefundTransactionUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	refundRepo port.RefundRepository,
	paymentGateway port.PaymentGatewayPort,
	idempotencyStore idempotency.IdempotencyStore,
) port.RefundTransactionUsecase {
	return &refundTransactionUsecase{
		txManager:        txManager,
		transactionRepo:  transactionRepo,
		refundRepo:       refundRepo,
		paymentGateway:   paymentGateway,
		idempotencyStore: idempotencyStore,
	}
}

func (u *refundTransactionUsecase) Execute(ctx context.Context, req dto.RefundRequest) (*dto.RefundResult, error) {
	// Simple Idempotency logic
	if req.IdempotencyKey != "" && u.idempotencyStore != nil {
		if record, err := u.idempotencyStore.Get(ctx, req.IdempotencyKey); err == nil && record != nil {
			var cachedRes dto.RefundResult
			if err := json.Unmarshal(record.Response, &cachedRes); err == nil {
				return &cachedRes, nil
			}
		}
	}

	trx, err := u.transactionRepo.FindByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	if trx.Status != domain.StatusSettlement {
		return nil, domain.NewValidationError("transaction is not settled")
	}

	if trx.Amount < req.Amount {
		return nil, domain.NewValidationError("refund amount exceeds transaction amount")
	}

	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}

	refund := &domain.Refund{
		ID:            domain.GenerateRefundID(time.Now()),
		TransactionID: trx.ID,
		Amount:        req.Amount,
		Reason:        reason,
		Status:        domain.RefundStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save pending refund
	err = u.txManager.WithTx(ctx, func(tx port.Tx) error {
		return u.refundRepo.Save(ctx, tx, refund)
	})
	if err != nil {
		return nil, fmt.Errorf("refundTransactionUsecase: failed to save pending refund: %w", err)
	}

	// Call Gateway
	updatedRefund, err := u.paymentGateway.RefundTransaction(ctx, trx.OrderID, refund)
	if err != nil {
		refund.Status = domain.RefundStatusFailed
	} else {
		refund.Status = updatedRefund.Status
		refund.MidtransRefundKey = updatedRefund.MidtransRefundKey
		refund.MidtransResponse = updatedRefund.MidtransResponse
	}

	// Update refund status
	_ = u.txManager.WithTx(ctx, func(tx port.Tx) error {
		return u.refundRepo.Update(ctx, tx, refund)
	})

	if err != nil {
		return nil, fmt.Errorf("refundTransactionUsecase: gateway error: %w", err)
	}

	res := &dto.RefundResult{
		RefundID:      refund.ID,
		TransactionID: refund.TransactionID,
		Amount:        refund.Amount,
		Status:        string(refund.Status),
		Reason:        refund.Reason,
	}

	if req.IdempotencyKey != "" && u.idempotencyStore != nil {
		respBytes, _ := json.Marshal(res)
		_ = u.idempotencyStore.Set(ctx, req.IdempotencyKey, idempotency.IdempotencyRecord{
			Key:        req.IdempotencyKey,
			StatusCode: 201,
			Response:   respBytes,
		}, 24*time.Hour)
	}

	return res, nil
}
