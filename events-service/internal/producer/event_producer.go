package producer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/segmentio/kafka-go"
)

const (
	maxRetries   = 3
	retryDelay   = 500 * time.Millisecond
	writeTimeout = 10 * time.Second
)

type EventProducer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewEventProducer(brokers []string, topic string, logger *slog.Logger) *EventProducer {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        false, // Synchronous for reliable delivery
		},
		logger: logger,
	}
}

func (p *EventProducer) publishWithRetry(ctx context.Context, eventType string, data interface{}, key string) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	wsEvent := models.WebSocketEvent{
		Type: eventType,
		Data: dataBytes,
	}

	eventBytes, err := json.Marshal(wsEvent)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: eventBytes,
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
				"event_type", eventType,
				"attempt", i+1,
				"max_attempts", maxRetries,
				"error", lastErr,
			)
			time.Sleep(retryDelay * time.Duration(i+1))
		}
	}

	p.logger.Error("Kafka publish failed after retries",
		"event_type", eventType,
		"attempts", maxRetries,
		"error", lastErr,
	)
	return lastErr
}

func (p *EventProducer) BroadcastRSVP(event models.EventRSVPEvent) {
	if err := p.publishWithRetry(context.Background(), "EVENT_RSVP_UPDATE", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish RSVP event", "error", err)
	}
}

func (p *EventProducer) PublishEventUpdated(ctx context.Context, event models.EventUpdatedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_UPDATED", event, event.ID); err != nil {
		p.logger.Error("Failed to publish EventUpdated event", "error", err)
	}
}

func (p *EventProducer) PublishEventDeleted(ctx context.Context, event models.EventDeletedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_DELETED", event, event.ID); err != nil {
		p.logger.Error("Failed to publish EventDeleted event", "error", err)
	}
}

func (p *EventProducer) PublishPostCreated(ctx context.Context, event models.EventPostCreatedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_POST_CREATED", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish PostCreated event", "error", err)
	}
}

func (p *EventProducer) PublishPostReaction(ctx context.Context, event models.EventPostReactionEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_POST_REACTION", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish PostReaction event", "error", err)
	}
}

func (p *EventProducer) PublishInvitationUpdated(ctx context.Context, event models.EventInvitationUpdatedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_INVITATION_UPDATED", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish InvitationUpdated event", "error", err)
	}
}

func (p *EventProducer) PublishCoHostAdded(ctx context.Context, event models.EventCoHostAddedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_COHOST_ADDED", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish CoHostAdded event", "error", err)
	}
}

func (p *EventProducer) PublishCoHostRemoved(ctx context.Context, event models.EventCoHostRemovedEvent) {
	if err := p.publishWithRetry(ctx, "EVENT_COHOST_REMOVED", event, event.EventID); err != nil {
		p.logger.Error("Failed to publish CoHostRemoved event", "error", err)
	}
}

func (p *EventProducer) Close() error {
	return p.writer.Close()
}
