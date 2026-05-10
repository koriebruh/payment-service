package usecase

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type handleMidtransWebhookUsecase struct {
	txManager       port.TxManager
	transactionRepo port.TransactionRepository
	webhookLogRepo  port.WebhookLogRepository
	outboxRepo      port.OutboxRepository
	serverKey       string
}

func NewHandleMidtransWebhookUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	webhookLogRepo port.WebhookLogRepository,
	outboxRepo port.OutboxRepository,
	serverKey string,
) port.HandleMidtransWebhookUsecase {
	return &handleMidtransWebhookUsecase{
		txManager:       txManager,
		transactionRepo: transactionRepo,
		webhookLogRepo:  webhookLogRepo,
		outboxRepo:      outboxRepo,
		serverKey:       serverKey,
	}
}

func (u *handleMidtransWebhookUsecase) Execute(ctx context.Context, req dto.WebhookPayload) error {
	ctx, span := otel.Tracer("payment-service/usecase").Start(ctx, "handleMidtransWebhookUsecase.Execute")
	defer span.End()

	// 1. Signature validation
	isValid := u.validateSignature(req.OrderID, req.StatusCode, req.GrossAmount, req.SignatureKey)
	signatureValidStr := "invalid"
	if isValid {
		signatureValidStr = "valid"
	}

	// Create log object early
	webhookLog := &domain.WebhookLog{
		ID:                    uuid.NewString(),
		MidtransTransactionID: &req.TransactionID,
		MidtransStatus:        &req.TransactionStatus,
		FraudStatus:           &req.FraudStatus,
		RawPayload:            req.RawPayload,
		SignatureValid:        signatureValidStr,
		ReceivedAt:            time.Now(),
	}

	if !isValid {
		slog.Warn("webhook signature invalid",
			"order_id", req.OrderID,
			"midtrans_txn_id", req.TransactionID,
		)
		if logErr := u.txManager.WithTx(ctx, func(tx port.Tx) error {
			return u.webhookLogRepo.Save(ctx, tx, webhookLog)
		}); logErr != nil {
			slog.Error("failed to save invalid webhook log", "error", logErr)
		}
		return domain.ErrInvalidSignature
	}

	// 2. Process logic inside Tx — pessimistic lock on transaction row to prevent race condition
	err := u.txManager.WithTx(ctx, func(tx port.Tx) error {
		// Idempotency check
		existingLog, err := u.webhookLogRepo.FindByMidtransTransactionIDAndStatus(ctx, req.TransactionID, req.TransactionStatus)
		if err == nil && existingLog != nil {
			slog.Info("webhook already processed (idempotent skip)",
				"order_id", req.OrderID,
				"midtrans_status", req.TransactionStatus,
			)
			return nil
		}

		// Pessimistic lock — SELECT ... FOR UPDATE
		trx, err := u.transactionRepo.FindByOrderIDForUpdate(ctx, tx, req.OrderID)
		if err != nil {
			return fmt.Errorf("find transaction for update: %w", err)
		}

		webhookLog.TransactionID = &trx.ID

		prevStatus := trx.Status
		u.updateTransactionStatus(trx, req.TransactionStatus, req.FraudStatus)

		slog.Info("webhook status transition",
			"order_id", req.OrderID,
			"prev_status", string(prevStatus),
			"new_status", string(trx.Status),
			"midtrans_status", req.TransactionStatus,
		)

		// Capture payment details from webhook payload
		trx.MidtransResponse = []byte(req.RawPayload)

		if err := u.transactionRepo.Update(ctx, tx, trx); err != nil {
			return fmt.Errorf("update transaction: %w", err)
		}

		if err := u.webhookLogRepo.Save(ctx, tx, webhookLog); err != nil {
			return fmt.Errorf("save webhook log: %w", err)
		}

		// If settled, save outbox event (atomic with the update — GEMINI.md rule 8)
		if trx.Status == domain.StatusSettlement {
			eventEnvelope := u.buildPaymentSettledEvent(trx)
			outboxEvent := &domain.OutboxEvent{
				ID:            uuid.NewString(),
				AggregateType: eventEnvelope.AggregateType,
				AggregateID:   eventEnvelope.AggregateID,
				EventType:     eventEnvelope.EventType,
				Payload:       eventEnvelope.Payload,
				Status:        domain.OutboxStatusPending,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			if err := u.outboxRepo.Save(ctx, tx, outboxEvent); err != nil {
				return fmt.Errorf("save outbox event: %w", err)
			}
			slog.Info("outbox event queued",
				"order_id", req.OrderID,
				"event_type", eventEnvelope.EventType,
			)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("handleMidtransWebhookUsecase.Execute: %w", err)
	}

	return nil
}

func (u *handleMidtransWebhookUsecase) validateSignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	payload := orderID + statusCode + grossAmount + u.serverKey
	hash := sha512.Sum512([]byte(payload))
	hashStr := hex.EncodeToString(hash[:])
	return hashStr == signatureKey
}

func (u *handleMidtransWebhookUsecase) updateTransactionStatus(trx *domain.Transaction, midtransStatus, fraudStatus string) {
	now := time.Now()
	switch midtransStatus {
	case "capture":
		if fraudStatus == "challenge" {
			trx.Status = domain.StatusPending
		} else if fraudStatus == "accept" {
			trx.Status = domain.StatusSettlement
			trx.PaidAt = &now
		}
	case "settlement":
		trx.Status = domain.StatusSettlement
		trx.PaidAt = &now
	case "cancel", "deny":
		trx.Status = domain.StatusCancel
	case "expire":
		trx.Status = domain.StatusExpire
	case "pending":
		trx.Status = domain.StatusPending
	}
	trx.UpdatedAt = now
}

func (u *handleMidtransWebhookUsecase) buildPaymentSettledEvent(trx *domain.Transaction) domain.EventEnvelope {
	paidAtStr := ""
	if trx.PaidAt != nil {
		paidAtStr = trx.PaidAt.Format(time.RFC3339)
	}

	payload := domain.PaymentSettledPayload{
		TransactionID: trx.ID,
		OrderID:       trx.OrderID,
		Amount:        trx.Amount,
		Currency:      trx.Currency,
		PaymentMethod: trx.PaymentMethodID,
		PaidAt:        paidAtStr,
	}

	payloadBytes, _ := json.Marshal(payload)
	traceID := ""
	if trx.CorrelationID != nil {
		traceID = *trx.CorrelationID
	}

	return domain.EventEnvelope{
		EventID:       uuid.NewString(),
		EventType:     "PAYMENT_SETTLED",
		SchemaVersion: "1.0.0",
		AggregateID:   trx.ID,
		AggregateType: "transaction",
		ServiceSource: "payment-service",
		TraceID:       traceID,
		CorrelationID: traceID,
		OccurredAt:    time.Now().UTC(),
		PublishedAt:   time.Now().UTC(),
		Payload:       payloadBytes,
	}
}
