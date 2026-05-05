package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/response"
)

type PaymentMethodHandler struct {
	getUsecase port.GetActivePaymentMethodsUsecase
	factory    *response.ApiResponseFactory
}

func NewPaymentMethodHandler(usecase port.GetActivePaymentMethodsUsecase, factory *response.ApiResponseFactory) *PaymentMethodHandler {
	return &PaymentMethodHandler{
		getUsecase: usecase,
		factory:    factory,
	}
}

func (h *PaymentMethodHandler) GetActiveMethods(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)

	results, err := h.getUsecase.Execute(c.Context())
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	return c.Status(fiber.StatusOK).JSON(h.factory.Success(requestID, "PAYMENT_METHODS_FETCHED", "success fetch payment methods", results))
}
