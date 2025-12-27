package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/integration"
	"github.com/MuhibNayem/connectify-v2/shared-entity/events"
	pkgkafka "github.com/MuhibNayem/connectify-v2/shared-entity/kafka"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserConsumer struct {
	reader      *kafka.Reader
	repo        *integration.UserLocalRepository
	dlqProducer *pkgkafka.DLQProducer
	logger      *slog.Logger
}

func NewUserConsumer(brokers []string, topic string, groupID string, repo *integration.UserLocalRepository, dlq *pkgkafka.DLQProducer, logger *slog.Logger) *UserConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MaxBytes: 10e6, // 10MB
	})
	if logger == nil {
		logger = slog.Default()
	}
	return &UserConsumer{
		reader:      r,
		repo:        repo,
		dlqProducer: dlq,
		logger:      logger,
	}
}

func (c *UserConsumer) Start(ctx context.Context) {
	c.logger.Info("Starting UserConsumer", "topic", c.reader.Config().Topic)
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			c.logger.Warn("UserConsumer stopped reading", "error", err)
			break
		}

		// Robust processing with retries
		maxRetries := 3
		var processErr error

		for i := 0; i < maxRetries; i++ {
			processErr = c.handleMessage(ctx, m.Value)
			if processErr == nil {
				break
			}

			if i < maxRetries-1 {
				c.logger.Warn("Error processing user event, retrying",
					"attempt", i+1,
					"max_attempts", maxRetries,
					"error", processErr,
				)
				time.Sleep(time.Second * time.Duration(i+1))
			}
		}

		if processErr != nil {
			c.logger.Error("Failed to process user event after retries, sending to DLQ",
				"attempts", maxRetries,
				"error", processErr,
			)
			if err := c.dlqProducer.PublishDeadLetter(ctx, c.reader.Config().Topic, m.Value, processErr); err != nil {
				c.logger.Error("Failed to send to DLQ", "error", err)
			}
		}
	}
}

func (c *UserConsumer) handleMessage(ctx context.Context, value []byte) error {
	var event events.UserUpdatedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return err // Send to DLQ
	}

	objID, err := primitive.ObjectIDFromHex(event.UserID)
	if err != nil {
		return err // Send to DLQ
	}

	return c.processMessage(ctx, objID, event.Username, event.FullName, event.Avatar, event.DateOfBirth)
}

func (c *UserConsumer) processMessage(ctx context.Context, userID primitive.ObjectID, username, fullName, avatar string, dob *time.Time) error {
	user := &integration.EventUser{
		ID:          userID,
		Username:    username,
		FullName:    fullName,
		Avatar:      avatar,
		DateOfBirth: dob,
	}

	return c.repo.UpsertUser(ctx, user)
}
