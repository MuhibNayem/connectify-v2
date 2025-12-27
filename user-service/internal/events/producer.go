package events

import (
	"context"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

const (
	maxRetries   = 3
	retryDelay   = 500 * time.Millisecond
	writeTimeout = 10 * time.Second
)

type EventProducer struct {
	writer *kafka.Writer
	topic  string
	logger *slog.Logger
}

func NewEventProducer(brokers []string, topic string, logger *slog.Logger) *EventProducer {
	if logger == nil {
		logger = slog.Default()
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    100,
		BatchBytes:   1024 * 1024,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Compression:  compress.Snappy,
		Async:        false, // Synchronous for reliable delivery
	}

	return &EventProducer{writer: w, topic: topic, logger: logger}
}

// Produce publishes a message with retry logic
func (p *EventProducer) Produce(ctx context.Context, key, value []byte) error {
	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
		lastErr = p.writer.WriteMessages(writeCtx, msg)
		cancel()

		if lastErr == nil {
			return nil
		}

		if i < maxRetries-1 {
			p.logger.Warn("Kafka publish failed, retrying",
				"topic", p.topic,
				"attempt", i+1,
				"max_attempts", maxRetries,
				"error", lastErr,
			)
			time.Sleep(retryDelay * time.Duration(i+1))
		}
	}

	p.logger.Error("Kafka publish failed after retries",
		"topic", p.topic,
		"attempts", maxRetries,
		"error", lastErr,
	)
	return lastErr
}

func (p *EventProducer) Close() error {
	return p.writer.Close()
}
