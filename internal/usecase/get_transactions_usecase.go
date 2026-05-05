package usecase

import (
	"context"

	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type getTransactionsUsecase struct {
	transactionRepo port.TransactionRepository
}

func NewGetTransactionsUsecase(transactionRepo port.TransactionRepository) port.GetTransactionsUsecase {
	return &getTransactionsUsecase{
		transactionRepo: transactionRepo,
	}
}

func (u *getTransactionsUsecase) Execute(ctx context.Context, req dto.ListTransactionsRequest) (*dto.TransactionListResult, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	transactions, totalCount, err := u.transactionRepo.FindAll(ctx, req.CustomerID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	var data []*dto.TransactionStatusResult
	for _, trx := range transactions {
		data = append(data, &dto.TransactionStatusResult{
			TransactionID:   trx.ID,
			OrderID:         trx.OrderID,
			Status:          string(trx.Status),
			Amount:          trx.Amount,
			Currency:        trx.Currency,
			PaymentMethodID: trx.PaymentMethodID,
			PaidAt:          trx.PaidAt,
			ExpiredAt:       trx.ExpiredAt,
		})
	}

	return &dto.TransactionListResult{
		Data:       data,
		TotalCount: totalCount,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}, nil
}
