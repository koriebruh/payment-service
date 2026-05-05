package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/koriebruh/payment-service/internal/adapter/handler/dto"
	"github.com/koriebruh/payment-service/internal/domain"
	usecase_dto "github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/response"
)

type WebhookHandler struct {
	webhookUsecase port.HandleMidtransWebhookUsecase
	factory        *response.ApiResponseFactory
}

func NewWebhookHandler(usecase port.HandleMidtransWebhookUsecase, factory *response.ApiResponseFactory) *WebhookHandler {
	return &WebhookHandler{
		webhookUsecase: usecase,
		factory:        factory,
	}
}

func (h *WebhookHandler) MidtransCallback(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)

	var reqDTO dto.WebhookRequestDTO
	if err := c.BodyParser(&reqDTO); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("invalid request body")),
		)
	}

	rawPayload := string(c.Body())

	req := usecase_dto.WebhookPayload{
		TransactionID:     reqDTO.TransactionID,
		OrderID:           reqDTO.OrderID,
		GrossAmount:       reqDTO.GrossAmount,
		StatusCode:        reqDTO.StatusCode,
		FraudStatus:       reqDTO.FraudStatus,
		TransactionStatus: reqDTO.TransactionStatus,
		SignatureKey:      reqDTO.SignatureKey,
		RawPayload:        rawPayload,
	}

	err := h.webhookUsecase.Execute(c.Context(), req)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			if errors.Is(err, domain.ErrInvalidSignature) {
				// Midtrans documentation says we should return 200 even for invalid signature
				// to stop midtrans from retrying, but logging it as unauthorized internally.
				// For safety, we return 401 as an example unless stated otherwise
				return c.Status(fiber.StatusOK).JSON(h.factory.Error(requestID, appErr))
			}
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		// Return 500 so midtrans retries
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	return c.Status(fiber.StatusOK).JSON(h.factory.SuccessNoData(requestID, "WEBHOOK_PROCESSED", "webhook processed successfully"))
}
