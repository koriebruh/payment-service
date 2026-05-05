package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/domain"
	"github.com/koriebruh/payment-service/internal/usecase/port"
)

type kafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(cfg *config.Config) port.EventPublisher {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	return &kafkaPublisher{
		writer: writer,
	}
}

func (p *kafkaPublisher) Publish(ctx context.Context, topic string, event domain.EventEnvelope) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event envelope: %w", err)
	}

	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(event.AggregateID),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	return nil
}
