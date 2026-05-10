package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type OutboxWorker struct {
	outboxRepo     port.OutboxRepository
	eventPublisher port.EventPublisher
	txManager      port.TxManager
	cfg            *config.Config
	pollInterval   time.Duration
}

func NewOutboxWorker(
	outboxRepo port.OutboxRepository,
	eventPublisher port.EventPublisher,
	txManager port.TxManager,
	cfg *config.Config,
	pollInterval time.Duration,
) *OutboxWorker {
	return &OutboxWorker{
		outboxRepo:     outboxRepo,
		eventPublisher: eventPublisher,
		txManager:      txManager,
		cfg:            cfg,
		pollInterval:   pollInterval,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("OutboxWorker stopping")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *OutboxWorker) processPendingEvents(ctx context.Context) {
	if err := w.txManager.WithTx(ctx, func(tx port.Tx) error {
		events, err := w.outboxRepo.FindPendingEvents(ctx, tx, 50)
		if err != nil {
			slog.Error("Failed to fetch pending outbox events", "error", err)
			return err
		}

		if len(events) == 0 {
			return nil
		}

		var publishedIDs []string

		for _, evt := range events {
			var envelope domain.EventEnvelope
			if err := json.Unmarshal(evt.Payload, &envelope); err != nil {
				slog.Error("Failed to unmarshal event payload", "event_id", evt.ID, "error", err)
				if markErr := w.outboxRepo.MarkAsFailed(ctx, tx, evt.ID); markErr != nil {
					slog.Error("Failed to mark event as failed", "event_id", evt.ID, "error", markErr)
				}
				continue
			}

			// Publish to Kafka
			if err := w.eventPublisher.Publish(ctx, w.cfg.Kafka.Topic, envelope); err != nil {
				slog.Error("Failed to publish event", "event_id", evt.ID, "error", err)
				continue // Retry next tick, lock will be released on commit
			}

			publishedIDs = append(publishedIDs, evt.ID)
		}

		// Batch update status to published
		if len(publishedIDs) > 0 {
			if err := w.outboxRepo.MarkAsPublished(ctx, tx, publishedIDs); err != nil {
				slog.Error("Failed to mark events as published", "error", err)
				return err
			}
		}

		return nil
	}); err != nil {
		slog.Error("OutboxWorker processPendingEvents tx error", "error", err)
	}
}
