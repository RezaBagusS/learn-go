package server

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"my-microservices/notification-service/config"
	"my-microservices/notification-service/helper"
	"my-microservices/notification-service/internal/client"
	"my-microservices/notification-service/internal/handler"
	"my-microservices/notification-service/internal/repository"
	"my-microservices/notification-service/internal/service"
)

type Server struct {
	db      *sql.DB
	readers map[string]*kafka.Reader
	cfg     *config.Config
}

func NewServer(db *sql.DB, readers map[string]*kafka.Reader, cfg *config.Config) *Server {
	return &Server{
		db:      db,
		readers: readers,
		cfg:     cfg,
	}
}

func (s *Server) Run() {

	notifRepo := repository.NewNotificationRepository(s.db)
	partnerCli := client.NewPartnerClient(
		s.cfg.Callback.TimeoutSeconds,
		s.cfg.Callback.MaxRetries,
		helper.Log,
	)

	notifSvc := service.NewNotificationService(notifRepo, partnerCli, helper.Log)
	kafkaHandler := handler.NewKafkaHandler(s.readers, notifSvc, helper.Log)

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		helper.Log.Info("Received shutdown signal", zap.String("signal", sig.String()))

		for t, r := range s.readers {
			if err := r.Close(); err != nil {
				helper.Log.Error("Failed to close reader", zap.String("topic", t), zap.Error(err))
			}
		}
		cancel()
	}()

	helper.Log.Info("notification-service started")

	if err := kafkaHandler.Start(ctx); err != nil {
		helper.Log.Error("Kafka handler exited with error", zap.Error(err))
	}

	helper.Log.Info("notification-service stopped gracefully")
}
