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

type FriendshipConsumer struct {
	reader         *kafka.Reader
	friendshipRepo *integration.FriendshipLocalRepository
	userRepo       *integration.UserLocalRepository
	dlqProducer    *pkgkafka.DLQProducer
	logger         *slog.Logger
}

func NewFriendshipConsumer(brokers []string, topic, groupID string, fr *integration.FriendshipLocalRepository, ur *integration.UserLocalRepository, dlq *pkgkafka.DLQProducer, logger *slog.Logger) *FriendshipConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	if logger == nil {
		logger = slog.Default()
	}

	return &FriendshipConsumer{
		reader:         reader,
		friendshipRepo: fr,
		userRepo:       ur,
		dlqProducer:    dlq,
		logger:         logger,
	}
}

func (c *FriendshipConsumer) Start(ctx context.Context) {
	defer c.reader.Close()

	c.logger.Info("FriendshipConsumer started", "topic", c.reader.Config().Topic)

	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			c.logger.Warn("Error reading message", "error", err)
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
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
				c.logger.Warn("Error processing friendship event, retrying",
					"attempt", i+1,
					"max_attempts", maxRetries,
					"error", processErr,
				)
				time.Sleep(time.Second * time.Duration(i+1))
			}
		}

		if processErr != nil {
			c.logger.Error("Failed to process friendship event after retries, sending to DLQ",
				"attempts", maxRetries,
				"error", processErr,
			)
			if err := c.dlqProducer.PublishDeadLetter(ctx, c.reader.Config().Topic, m.Value, processErr); err != nil {
				c.logger.Error("Failed to send to DLQ", "error", err)
			}
		}
	}
}

func (c *FriendshipConsumer) handleMessage(ctx context.Context, value []byte) error {
	var event events.FriendshipEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return err // Permanent error, but we'll send to DLQ to analyze malformed data
	}
	return c.processEvent(ctx, event)
}

func (c *FriendshipConsumer) processEvent(ctx context.Context, event events.FriendshipEvent) error {
	reqID, err := primitive.ObjectIDFromHex(event.RequesterID)
	if err != nil {
		return err
	}
	recID, err := primitive.ObjectIDFromHex(event.ReceiverID)
	if err != nil {
		return err
	}

	friendship := &integration.EventFriendship{
		RequesterID: reqID,
		ReceiverID:  recID,
		Status:      event.Status,
		CreatedAt:   event.Timestamp,
	}

	switch event.Action {
	case "accept": // Accepted
		// Upsert Friendship
		if err := c.friendshipRepo.UpsertFriendship(ctx, friendship); err != nil {
			return err
		}
		// Update Users (Add Friend)
		_ = c.userRepo.AddFriend(ctx, reqID, recID)
		_ = c.userRepo.AddFriend(ctx, recID, reqID)

	case "remove", "block": // Removed or Blocked
		// Remove Friendship
		if err := c.friendshipRepo.RemoveFriendship(ctx, reqID, recID); err != nil {
			return err
		}
		// Update Users (Remove Friend)
		_ = c.userRepo.RemoveFriend(ctx, reqID, recID)
		_ = c.userRepo.RemoveFriend(ctx, recID, reqID)

	case "request", "reject", "unblock":
		// Just upsert status
		if event.Action != "unblock" {
			_ = c.friendshipRepo.UpsertFriendship(ctx, friendship)
		}
	}

	return nil
}
