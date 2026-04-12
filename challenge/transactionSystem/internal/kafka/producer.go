package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	writers map[string]*kafka.Writer
	brokers []string // tambahkan ini
	logger  *zap.Logger
}

func NewProducer(brokers []string, logger *zap.Logger) *Producer {
	return &Producer{
		writers: make(map[string]*kafka.Writer),
		logger:  logger,
		brokers: brokers,
	}
}

func (p *Producer) getWriter(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		WriteTimeout: 10 * time.Second,
		// Async: false → tunggu ack sebelum return, lebih aman untuk financial transaction
		Async: false,
	}

	p.writers[topic] = w
	return w
}

// Publish mengirim event ke topic tertentu dengan key sebagai routing key
func (p *Producer) Publish(ctx context.Context, topic, key string, payload any) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("kafka producer: failed to marshal payload: %w", err)
	}

	writer := p.getWriter(topic)

	if err := writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	}); err != nil {
		p.logger.Error("kafka producer: failed to publish",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("kafka producer: failed to publish to topic %s: %w", topic, err)
	}

	p.logger.Info("kafka producer: message published",
		zap.String("topic", topic),
		zap.String("key", key),
	)

	return nil
}

// Close menutup semua writer secara graceful
func (p *Producer) Close() {
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			p.logger.Error("kafka producer: failed to close writer",
				zap.String("topic", topic),
				zap.Error(err),
			)
		}
	}
}
