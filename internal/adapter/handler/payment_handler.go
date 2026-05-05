package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/koriebruh/payment-service/internal/adapter/handler/dto"
	"github.com/koriebruh/payment-service/internal/domain"
	usecase_dto "github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/response"
)

type PaymentHandler struct {
	chargeUsecase port.ChargeTransactionUsecase
	validator     *validator.Validate
	factory       *response.ApiResponseFactory
}

func NewPaymentHandler(usecase port.ChargeTransactionUsecase, validate *validator.Validate, factory *response.ApiResponseFactory) *PaymentHandler {
	return &PaymentHandler{
		chargeUsecase: usecase,
		validator:     validate,
		factory:       factory,
	}
}

func (h *PaymentHandler) Charge(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)
	traceID, _ := c.Locals("trace_id").(string)

	var reqDTO dto.ChargeRequestDTO
	if err := c.BodyParser(&reqDTO); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("invalid request body")),
		)
	}

	reqDTO.IdempotencyKey = c.Get("X-Idempotency-Key")
	if reqDTO.IdempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("missing X-Idempotency-Key header")),
		)
	}

	if err := h.validator.Struct(reqDTO); err != nil {
		fields := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			fields[err.Field()] = err.Tag()
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			h.factory.ValidationError(requestID, fields),
		)
	}

	req := usecase_dto.ChargeRequest{
		CustomerID:      reqDTO.CustomerID,
		PaymentMethodID: reqDTO.PaymentMethodID,
		Amount:          reqDTO.Amount,
		Currency:        reqDTO.Currency,
		IdempotencyKey:  reqDTO.IdempotencyKey,
		RequestID:       requestID,
		TraceID:         traceID,
	}

	result, err := h.chargeUsecase.Execute(c.Context(), req)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	respDTO := dto.ChargeResponseDTO{
		TransactionID: result.TransactionID,
		OrderID:       result.OrderID,
		Status:        result.Status,
		PaymentURL:    result.PaymentURL,
		SnapToken:     result.SnapToken,
	}

	return c.Status(fiber.StatusCreated).JSON(h.factory.Success(requestID, "CHARGE_SUCCESS", "charge initiated", respDTO))
}
