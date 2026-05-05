package usecase

import (
	"context"
	"encoding/json"

	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type getTransactionStatusUsecase struct {
	transactionRepo port.TransactionRepository
	refundRepo      port.RefundRepository
}

func NewGetTransactionStatusUsecase(transactionRepo port.TransactionRepository, refundRepo port.RefundRepository) port.GetTransactionStatusUsecase {
	return &getTransactionStatusUsecase{
		transactionRepo: transactionRepo,
		refundRepo:      refundRepo,
	}
}

func (u *getTransactionStatusUsecase) Execute(ctx context.Context, orderID string) (*dto.TransactionStatusResult, error) {
	trx, err := u.transactionRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Fetch Refunds
	refundsData, err := u.refundRepo.FindByTransactionID(ctx, trx.ID)
	if err != nil {
		return nil, err
	}

	var refunds []dto.RefundDetail
	for _, r := range refundsData {
		refunds = append(refunds, dto.RefundDetail{
			RefundID:  r.ID,
			Amount:    r.Amount,
			Reason:    r.Reason,
			Status:    string(r.Status),
			CreatedAt: r.CreatedAt,
		})
	}

	if refunds == nil {
		refunds = make([]dto.RefundDetail, 0)
	}

	// Parse Payment Details
	var paymentDetails map[string]interface{}
	if len(trx.MidtransResponse) > 0 {
		_ = json.Unmarshal(trx.MidtransResponse, &paymentDetails)
		// Clean up unnecessary raw fields if desired, or return as is
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
		PaymentDetails:  paymentDetails,
		Refunds:         refunds,
	}, nil
}
