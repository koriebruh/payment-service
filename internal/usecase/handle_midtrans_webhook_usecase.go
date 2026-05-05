package usecase

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/dto"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type handleMidtransWebhookUsecase struct {
	txManager       port.TxManager
	transactionRepo port.TransactionRepository
	webhookLogRepo  port.WebhookLogRepository
	outboxRepo      port.OutboxRepository
	cfg             *config.Config
}

func NewHandleMidtransWebhookUsecase(
	txManager port.TxManager,
	transactionRepo port.TransactionRepository,
	webhookLogRepo port.WebhookLogRepository,
	outboxRepo port.OutboxRepository,
	cfg *config.Config,
) port.HandleMidtransWebhookUsecase {
	return &handleMidtransWebhookUsecase{
		txManager:       txManager,
		transactionRepo: transactionRepo,
		webhookLogRepo:  webhookLogRepo,
		outboxRepo:      outboxRepo,
		cfg:             cfg,
	}
}

func (u *handleMidtransWebhookUsecase) Execute(ctx context.Context, req dto.WebhookPayload) error {
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
		// Log the invalid attempt and return error so we don't process further
		_ = u.txManager.WithTx(ctx, func(tx port.Tx) error {
			return u.webhookLogRepo.Save(ctx, tx, webhookLog)
		})
		return domain.ErrInvalidSignature
	}

	// 2. Process logic inside Tx
	err := u.txManager.WithTx(ctx, func(tx port.Tx) error {
		// Idempotency check: Have we processed this midtrans_transaction_id with this status before?
		existingLog, err := u.webhookLogRepo.FindByMidtransTransactionIDAndStatus(ctx, req.TransactionID, req.TransactionStatus)
		if err == nil && existingLog != nil {
			// Already processed, return nil to act as idempotent success
			return nil
		}

		// Find transaction
		trx, err := u.transactionRepo.FindByOrderID(ctx, req.OrderID)
		if err != nil {
			return err
		}

		webhookLog.TransactionID = &trx.ID

		// Update state machine
		u.updateTransactionStatus(trx, req.TransactionStatus, req.FraudStatus)
		
		// Capture payment details (VA, QRIS, etc.) from webhook payload
		trx.MidtransResponse = []byte(req.RawPayload)

		// Save updates
		if err := u.transactionRepo.Update(ctx, tx, trx); err != nil {
			return err
		}

		// Save webhook log
		if err := u.webhookLogRepo.Save(ctx, tx, webhookLog); err != nil {
			return err
		}

		// If settled, save outbox event
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
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("handleMidtransWebhookUsecase.Execute: %w", err)
	}

	return nil
}

func (u *handleMidtransWebhookUsecase) validateSignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	payload := orderID + statusCode + grossAmount + u.cfg.Midtrans.ServerKey
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
