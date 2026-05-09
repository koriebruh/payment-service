package usecase

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type getActivePaymentMethodsUsecase struct {
	pmRepo port.PaymentMethodRepository
}

func NewGetActivePaymentMethodsUsecase(pmRepo port.PaymentMethodRepository) port.GetActivePaymentMethodsUsecase {
	return &getActivePaymentMethodsUsecase{
		pmRepo: pmRepo,
	}
}

func (u *getActivePaymentMethodsUsecase) Execute(ctx context.Context) ([]*dto.PaymentMethodResult, error) {
	ctx, span := otel.Tracer("payment-service/usecase").Start(ctx, "getActivePaymentMethodsUsecase.Execute")
	defer span.End()

	methods, err := u.pmRepo.FindAllActive(ctx)
	if err != nil {
		return nil, err
	}

	var results []*dto.PaymentMethodResult
	for _, m := range methods {
		results = append(results, &dto.PaymentMethodResult{
			ID:   m.ID,
			Name: m.Name,
			Type: m.Type,
		})
	}

	return results, nil
}
