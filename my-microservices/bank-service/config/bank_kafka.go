package config

import (
	"my-microservices/bank-service/internal/kafka"
	"os"
	"strings"

	"go.uber.org/zap"
)

func ConnectKafka(logger *zap.Logger) *kafka.Producer {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = "localhost:9092" // fallback default
	}

	brokers := strings.Split(brokersEnv, ",")
	producer := kafka.NewProducer(brokers, logger)

	logger.Info("kafka producer initialized", zap.Strings("brokers", brokers))

	return producer
}
