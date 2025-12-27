package consumer

import (
	"context"
	"encoding/json"
	"log"
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
}

func NewUserConsumer(brokers []string, topic string, groupID string, repo *integration.UserLocalRepository, dlq *pkgkafka.DLQProducer) *UserConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MaxBytes: 10e6, // 10MB
	})
	return &UserConsumer{
		reader:      r,
		repo:        repo,
		dlqProducer: dlq,
	}
}

func (c *UserConsumer) Start(ctx context.Context) {
	log.Printf("Starting UserConsumer for topic %v", c.reader.Config().Topic)
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("UserConsumer stopped reading: %v", err)
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
				log.Printf("Error processing user event (attempt %d/%d): %v. Retrying...", i+1, maxRetries, processErr)
				time.Sleep(time.Second * time.Duration(i+1))
			}
		}

		if processErr != nil {
			log.Printf("CRITICAL: Failed to process user event after %d attempts. Sending to DLQ. Error: %v", maxRetries, processErr)
			if err := c.dlqProducer.PublishDeadLetter(ctx, c.reader.Config().Topic, m.Value, processErr); err != nil {
				log.Printf("FATAL: Failed to send to DLQ: %v", err)
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
	// Note: UpsertUser updates the whole document.
	return c.repo.UpsertUser(ctx, user)
}
