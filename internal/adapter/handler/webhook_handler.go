package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/koriebruh/payment-service/internal/adapter/handler/dto"
	"github.com/koriebruh/payment-service/internal/domain"
	usecase_dto "github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/logger"
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
	log := logger.FromContext(c.UserContext())

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

	log.Info("webhook received",
		"order_id", reqDTO.OrderID,
		"midtrans_status", reqDTO.TransactionStatus,
		"midtrans_txn_id", reqDTO.TransactionID,
	)

	err := h.webhookUsecase.Execute(c.Context(), req)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			if errors.Is(err, domain.ErrInvalidSignature) {
				log.Warn("webhook rejected: invalid signature",
					"order_id", reqDTO.OrderID,
					"midtrans_txn_id", reqDTO.TransactionID,
				)
				return c.Status(fiber.StatusOK).JSON(h.factory.Error(requestID, appErr))
			}
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		log.Error("webhook processing failed",
			"order_id", reqDTO.OrderID,
			"error", err.Error(),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	log.Info("webhook processed",
		"order_id", reqDTO.OrderID,
		"midtrans_status", reqDTO.TransactionStatus,
	)

	return c.Status(fiber.StatusOK).JSON(h.factory.SuccessNoData(requestID, "WEBHOOK_PROCESSED", "webhook processed successfully"))
}
