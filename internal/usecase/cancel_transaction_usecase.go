package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type cancelTransactionUsecase struct {
	txManager       port.TxManager
	transactionRepo port.TransactionRepository
	paymentGateway  port.PaymentGatewayPort
}

func NewCancelTransactionUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	paymentGateway port.PaymentGatewayPort,
) port.CancelTransactionUsecase {
	return &cancelTransactionUsecase{
		txManager:       txManager,
		transactionRepo: transactionRepo,
		paymentGateway:  paymentGateway,
	}
}

func (u *cancelTransactionUsecase) Execute(ctx context.Context, orderID string) error {
	trx, err := u.transactionRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if trx.Status != domain.StatusPending {
		return domain.NewValidationError("only pending transaction can be cancelled")
	}

	// Cancel on midtrans first
	err = u.paymentGateway.CancelTransaction(ctx, trx.OrderID)
	if err != nil {
		// Sometimes midtrans says 404 or something if it's already cancelled
		// but we still want to log the error if it fails for other reasons
		return fmt.Errorf("failed to cancel midtrans transaction: %w", err)
	}

	trx.Status = domain.StatusCancel
	trx.UpdatedAt = time.Now()

	err = u.txManager.WithTx(ctx, func(tx port.Tx) error {
		return u.transactionRepo.Update(ctx, tx, trx)
	})

	return err
}
