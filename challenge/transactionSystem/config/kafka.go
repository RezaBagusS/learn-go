package config

import (
	"os"
	"strings"

	"belajar-go/challenge/transactionSystem/internal/kafka"

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

func KafkaBrokers() []string {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		return []string{"localhost:9092"}
	}
	return strings.Split(brokersEnv, ",")
}
