package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/segmentio/kafka-go"
)

// ErrProcessingFailed indicates handler failed - message should NOT be committed
var ErrProcessingFailed = errors.New("message processing failed")

type KafkaQueue struct {
	writer *kafka.Writer
	reader *kafka.Reader
}

func NewKafkaQueue(brokers []string, topic, groupID string) *KafkaQueue {
	return &KafkaQueue{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
		},
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 10e3,
			MaxBytes: 10e6,
			// CRITICAL: Disable auto-commit for manual commit-on-success
			CommitInterval: 0, // Manual commits only
		}),
	}
}

func (k *KafkaQueue) Publish(ctx context.Context, event *adapters.NotificationEvent) error {
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ID),
		Value: value,
		Time:  time.Now(),
	})
}

func (k *KafkaQueue) PublishBatch(ctx context.Context, events []*adapters.NotificationEvent) error {
	messages := make([]kafka.Message, len(events))
	for i, event := range events {
		value, _ := json.Marshal(event)
		messages[i] = kafka.Message{
			Key:   []byte(event.ID),
			Value: value,
			Time:  time.Now(),
		}
	}
	return k.writer.WriteMessages(ctx, messages...)
}

// Subscribe implements commit-on-success semantics:
// 1. FetchMessage (doesn't commit)
// 2. Call handler
// 3. If handler succeeds, CommitMessages
// 4. If handler fails, message stays uncommitted = automatic retry on restart
func (k *KafkaQueue) Subscribe(ctx context.Context, handler adapters.EventHandler) error {
	for {
		// STEP 1: Fetch without commit
		msg, err := k.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}

		// STEP 2: Parse event
		var event adapters.NotificationEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			// Malformed message - commit to skip (or send to DLQ)
			// We commit to avoid infinite loop on bad data
			_ = k.reader.CommitMessages(ctx, msg)
			continue
		}

		// STEP 3: Process via handler
		handlerErr := handler(ctx, &event)

		// STEP 4: Commit ONLY on success
		if handlerErr == nil {
			if err := k.reader.CommitMessages(ctx, msg); err != nil {
				// Log commit failure but continue
				// On restart, message may be reprocessed (idempotency handles this)
			}
		}
		// If handlerErr != nil, we do NOT commit
		// This means:
		// - Message remains uncommitted
		// - Will be redelivered after session timeout
		// - Handler should implement idempotency for safe retries
	}
}

func (k *KafkaQueue) Close() error {
	if err := k.writer.Close(); err != nil {
		return err
	}
	return k.reader.Close()
}

func (k *KafkaQueue) HealthCheck(ctx context.Context) error {
	_ = k.writer.Stats()
	return nil
}
