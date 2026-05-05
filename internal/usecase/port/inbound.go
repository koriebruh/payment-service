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
