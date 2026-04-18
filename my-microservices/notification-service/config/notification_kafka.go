package config

import (
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func ConnectKafkaReaders(cfg KafkaConfig, log *zap.Logger) map[string]*kafka.Reader {
	readers := make(map[string]*kafka.Reader)
	topics := []string{"account.created", "transaction.created", "transaction.failed", "account.balance_updated"}

	for _, topic := range topics {
		readers[topic] = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{cfg.Brokers},
			GroupID:        cfg.GroupID,
			Topic:          topic,
			MinBytes:       10e3,
			MaxBytes:       10e6,
			ReadBackoffMin: 100 * time.Millisecond,
			ReadBackoffMax: 1 * time.Second,
		})
		log.Info("Kafka reader initialized", zap.String("topic", topic))
	}
	return readers
}
