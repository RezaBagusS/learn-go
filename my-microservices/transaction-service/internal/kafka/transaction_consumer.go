package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type HandlerFunc[T any] func(ctx context.Context, payload T) error

type Consumer[T any] struct {
	reader  *kafka.Reader
	logger  *zap.Logger
	topic   string
	groupID string
}

func NewConsumer[T any](brokers []string, topic, groupID string, logger *zap.Logger) *Consumer[T] {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})

	return &Consumer[T]{
		reader:  r,
		logger:  logger,
		topic:   topic,
		groupID: groupID,
	}
}

func (c *Consumer[T]) Start(ctx context.Context, handler HandlerFunc[T]) {
	go c.consume(ctx, handler)
}

func (c *Consumer[T]) consume(ctx context.Context, handler HandlerFunc[T]) {
	c.logger.Info("kafka consumer: started",
		zap.String("topic", c.topic),
		zap.String("group_id", c.groupID),
	)

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("kafka consumer: shutting down", zap.String("topic", c.topic))
				return
			}
			c.logger.Error("kafka consumer: failed to fetch message",
				zap.String("topic", c.topic),
				zap.Error(err),
			)
			continue
		}

		var payload T
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			c.logger.Error("kafka consumer: failed to unmarshal message",
				zap.String("topic", c.topic),
				zap.ByteString("raw", msg.Value),
				zap.Error(err),
			)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		if err := handler(ctx, payload); err != nil {
			c.logger.Error("kafka consumer: handler returned error",
				zap.String("topic", c.topic),
				zap.String("key", string(msg.Key)),
				zap.Error(err),
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("kafka consumer: failed to commit offset",
				zap.String("topic", c.topic),
				zap.Error(err),
			)
		}
	}
}

func (c *Consumer[T]) Close() error {
	return c.reader.Close()
}
