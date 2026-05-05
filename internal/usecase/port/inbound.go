package port

import (
	"context"

	"github.com/koriebruh/payment-service/internal/usecase/dto"
)

type ChargeTransactionUsecase interface {
	Execute(ctx context.Context, req dto.ChargeRequest) (*dto.ChargeResult, error)
}

type HandleMidtransWebhookUsecase interface {
	Execute(ctx context.Context, req dto.WebhookPayload) error
}

type GetTransactionStatusUsecase interface {
	Execute(ctx context.Context, orderID string) (*dto.TransactionStatusResult, error)
}

type RefundTransactionUsecase interface {
	Execute(ctx context.Context, req dto.RefundRequest) (*dto.RefundResult, error)
}

type GetActivePaymentMethodsUsecase interface {
	Execute(ctx context.Context) ([]*dto.PaymentMethodResult, error)
}
