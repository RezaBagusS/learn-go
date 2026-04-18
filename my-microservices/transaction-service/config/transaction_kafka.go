package config

import (
	"my-microservices/transaction-service/internal/kafka"
	"os"
	"strings"

	"go.uber.org/zap"
)

func ConnectKafka(logger *zap.Logger) *kafka.Producer {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = "localhost:9092"
	}

	brokers := strings.Split(brokersEnv, ",")
	producer := kafka.NewProducer(brokers, logger)

	logger.Info("kafka producer initialized (transaction-service)", zap.Strings("brokers", brokers))

	return producer
}
