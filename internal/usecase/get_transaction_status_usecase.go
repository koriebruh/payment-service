package usecase

import (
	"context"

	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type getTransactionStatusUsecase struct {
	transactionRepo port.TransactionRepository
}

func NewGetTransactionStatusUsecase(transactionRepo port.TransactionRepository) port.GetTransactionStatusUsecase {
	return &getTransactionStatusUsecase{
		transactionRepo: transactionRepo,
	}
}

func (u *getTransactionStatusUsecase) Execute(ctx context.Context, orderID string) (*dto.TransactionStatusResult, error) {
	trx, err := u.transactionRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return &dto.TransactionStatusResult{
		TransactionID:   trx.ID,
		OrderID:         trx.OrderID,
		Status:          string(trx.Status),
		Amount:          trx.Amount,
		Currency:        trx.Currency,
		PaymentMethodID: trx.PaymentMethodID,
		PaidAt:          trx.PaidAt,
		ExpiredAt:       trx.ExpiredAt,
	}, nil
}
