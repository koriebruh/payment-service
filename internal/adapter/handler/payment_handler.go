package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"github.com/koriebruh/payment-service/internal/adapter/handler/dto"
	"github.com/koriebruh/payment-service/internal/domain"
	usecase_dto "github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
	"github.com/koriebruh/payment-service/pkg/logger"
	"github.com/koriebruh/payment-service/pkg/response"
)

type PaymentHandler struct {
	chargeUsecase    port.ChargeTransactionUsecase
	getStatusUsecase port.GetTransactionStatusUsecase
	refundUsecase    port.RefundTransactionUsecase
	cancelUsecase    port.CancelTransactionUsecase
	getListUsecase   port.GetTransactionsUsecase
	validator        *validator.Validate
	factory          *response.ApiResponseFactory
}

func NewPaymentHandler(
	chargeUsecase port.ChargeTransactionUsecase,
	getStatusUsecase port.GetTransactionStatusUsecase,
	refundUsecase port.RefundTransactionUsecase,
	cancelUsecase port.CancelTransactionUsecase,
	getListUsecase port.GetTransactionsUsecase,
	validate *validator.Validate,
	factory *response.ApiResponseFactory,
) *PaymentHandler {
	return &PaymentHandler{
		chargeUsecase:    chargeUsecase,
		getStatusUsecase: getStatusUsecase,
		refundUsecase:    refundUsecase,
		cancelUsecase:    cancelUsecase,
		getListUsecase:   getListUsecase,
		validator:        validate,
		factory:          factory,
	}
}

func (h *PaymentHandler) Charge(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)
	traceID, _ := c.Locals("trace_id").(string)
	log := logger.FromContext(c.UserContext())

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
		log.Error("charge failed",
			"customer_id", reqDTO.CustomerID,
			"payment_method", reqDTO.PaymentMethodID,
			"amount", reqDTO.Amount,
			"error", err.Error(),
		)
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	log.Info("charge success",
		"transaction_id", result.TransactionID,
		"order_id", result.OrderID,
		"amount", reqDTO.Amount,
	)

	respDTO := dto.ChargeResponseDTO{
		TransactionID: result.TransactionID,
		OrderID:       result.OrderID,
		Status:        result.Status,
		PaymentURL:    result.PaymentURL,
		SnapToken:     result.SnapToken,
	}

	return c.Status(fiber.StatusCreated).JSON(h.factory.Success(requestID, "CHARGE_SUCCESS", "charge initiated", respDTO))
}

func (h *PaymentHandler) GetStatus(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)
	orderID := c.Params("order_id")

	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("order_id parameter is required")),
		)
	}

	result, err := h.getStatusUsecase.Execute(c.Context(), orderID)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	return c.Status(fiber.StatusOK).JSON(h.factory.Success(requestID, "FETCH_STATUS_SUCCESS", "success fetch transaction status", result))
}

// Struct for refund request parser
type RefundRequestPayload struct {
	Amount int64  `json:"amount" validate:"required,gt=0"`
	Reason string `json:"reason" validate:"required"`
}

func (h *PaymentHandler) Refund(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)
	log := logger.FromContext(c.UserContext())
	orderID := c.Params("order_id")

	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("order_id parameter is required")),
		)
	}

	var reqPayload RefundRequestPayload
	if err := c.BodyParser(&reqPayload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("invalid request body")),
		)
	}

	if err := h.validator.Struct(reqPayload); err != nil {
		fields := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			fields[err.Field()] = err.Tag()
		}
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			h.factory.ValidationError(requestID, fields),
		)
	}

	idempotencyKey := c.Get("X-Idempotency-Key")

	req := usecase_dto.RefundRequest{
		OrderID:        orderID,
		Amount:         reqPayload.Amount,
		Reason:         reqPayload.Reason,
		IdempotencyKey: idempotencyKey,
	}

	result, err := h.refundUsecase.Execute(c.Context(), req)
	if err != nil {
		log.Error("refund failed", "order_id", orderID, "amount", reqPayload.Amount, "error", err.Error())
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	log.Info("refund initiated", "order_id", orderID, "refund_id", result.RefundID, "amount", reqPayload.Amount)

	return c.Status(fiber.StatusCreated).JSON(h.factory.Success(requestID, "REFUND_INITIATED", "refund initiated successfully", result))
}

func (h *PaymentHandler) Cancel(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)
	log := logger.FromContext(c.UserContext())
	orderID := c.Params("order_id")

	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(
			h.factory.Error(requestID, domain.NewValidationError("order_id parameter is required")),
		)
	}

	err := h.cancelUsecase.Execute(c.Context(), orderID)
	if err != nil {
		log.Error("cancel failed", "order_id", orderID, "error", err.Error())
		var appErr *domain.AppError
		if errors.As(err, &appErr) {
			return c.Status(appErr.HTTPStatus).JSON(h.factory.Error(requestID, appErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	log.Info("transaction cancelled", "order_id", orderID)

	return c.Status(fiber.StatusOK).JSON(h.factory.SuccessNoData(requestID, "CANCEL_SUCCESS", "transaction cancelled successfully"))
}

func (h *PaymentHandler) GetTransactions(c *fiber.Ctx) error {
	requestID, _ := c.Locals("request_id").(string)

	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)
	customerID := c.Query("customer_id")

	req := usecase_dto.ListTransactionsRequest{
		CustomerID: customerID,
		Limit:      limit,
		Offset:     offset,
	}

	result, err := h.getListUsecase.Execute(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(h.factory.Error(requestID, domain.NewInternalError(err)))
	}

	return c.Status(fiber.StatusOK).JSON(h.factory.Success(requestID, "FETCH_TRANSACTIONS_SUCCESS", "success fetch transactions", result))
}
