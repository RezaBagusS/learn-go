package handler

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"my-microservices/notification-service/internal/service"
)

type KafkaHandler struct {
	readers map[string]*kafka.Reader
	svc     service.NotificationService
	log     *zap.Logger
}

func NewKafkaHandler(readers map[string]*kafka.Reader, svc service.NotificationService, log *zap.Logger) *KafkaHandler {
	return &KafkaHandler{
		readers: readers,
		svc:     svc,
		log:     log,
	}
}

func (h *KafkaHandler) Start(ctx context.Context) error {
	var wg sync.WaitGroup

	for topicName, reader := range h.readers {
		wg.Add(1)
		go func(topic string, r *kafka.Reader) {
			defer wg.Done()
			h.log.Info("Starting consumer for topic", zap.String("topic", topic))

			for {
				msg, err := r.ReadMessage(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
						h.log.Info("Consumer stopping gracefully", zap.String("topic", topic))
						return
					}
					h.log.Error("Failed to read message", zap.String("topic", topic), zap.Error(err))
					continue
				}

				err = h.svc.ProcessEvent(ctx, topic, msg.Value)
				if err != nil {
					h.log.Error("Failed to process event",
						zap.String("topic", topic),
						zap.ByteString("payload", msg.Value),
						zap.Error(err),
					)
				} else {
					h.log.Info("Message processed successfully",
						zap.String("topic", topic),
						zap.Int64("offset", msg.Offset),
					)
				}
			}
		}(topicName, reader)
	}

	wg.Wait()
	return nil
}
