package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Kafka    KafkaConfig
	Database DatabaseConfig
	Callback CallbackConfig
}

type AppConfig struct {
	Env  string
	Port string
}

type KafkaConfig struct {
	Brokers         string
	GroupID         string
	AutoOffsetReset string
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type CallbackConfig struct {
	TimeoutSeconds int
	MaxRetries     int
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	connMaxLifetime, _ := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	callbackTimeout, _ := strconv.Atoi(getEnv("CALLBACK_TIMEOUT_SECONDS", "30"))
	maxRetries, _ := strconv.Atoi(getEnv("CALLBACK_MAX_RETRIES", "3"))

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
		},
		Kafka: KafkaConfig{
			Brokers:         getEnv("KAFKA_BROKERS", "localhost:9092"),
			GroupID:         getEnv("KAFKA_GROUP_ID", "notification-service"),
			AutoOffsetReset: getEnv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "notification_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
		},
		Callback: CallbackConfig{
			TimeoutSeconds: callbackTimeout,
			MaxRetries:     maxRetries,
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
