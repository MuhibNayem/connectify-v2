package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/config"
	"github.com/MuhibNayem/connectify-v2/shared-entity/events"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

type EventProducer struct {
	cfg         *config.Config
	wsWriter    *kafka.Writer
	notifWriter *kafka.Writer
}

func NewEventProducer(cfg *config.Config) *EventProducer {
	return &EventProducer{
		cfg: cfg,
		wsWriter: &kafka.Writer{
			Addr:         kafka.TCP(cfg.KafkaBrokers...),
			Topic:        cfg.KafkaTopic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        true,
		},
		notifWriter: &kafka.Writer{
			Addr:         kafka.TCP(cfg.KafkaBrokers...),
			Topic:        cfg.NotificationTopic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        true,
		},
	}
}

func (p *EventProducer) PublishWebSocketEvent(ctx context.Context, userID string, eventType string, data interface{}) error {
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
		Key:   []byte(userID),
		Value: eventBytes,
		Time:  time.Now(),
	}

	return p.wsWriter.WriteMessages(ctx, msg)
}

func (p *EventProducer) PublishEvent(topic string, event models.WebSocketEvent) error {
	// topic arg is currently unused as we have dedicated writers,
	// but keeping signature generic for future extensibility if we add more writers.
	// We assume 'messages' topic implies wsWriter.

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Use SenderID or Random Key?
	// If we want ordering per sender, use Sender?
	// For now, let's use a random key or no key (round-robin) since it's a broadcast-like event.
	// Or, if event.Data has UserID, extract it? Too complex.
	// Let's use current time as key for random distribution or empty.

	msg := kafka.Message{
		// Key:   nil, // Round-robin
		Value: eventBytes,
		Time:  time.Now(),
	}

	return p.wsWriter.WriteMessages(context.Background(), msg)
}

func (p *EventProducer) PublishNotification(ctx context.Context, apiEvent *events.NotificationCreatedEvent) error {
	// Generate new ID if not present (although shared-entity usually relies on mongo ID)
	// Here we construct the event as expected by NotificationConsumer

	payload, err := json.Marshal(apiEvent)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(apiEvent.RecipientID.Hex()),
		Value: payload,
		Time:  time.Now(),
	}

	return p.notifWriter.WriteMessages(ctx, msg)
}

func (p *EventProducer) Close() {
	if err := p.wsWriter.Close(); err != nil {
		log.Printf("Error closing WS writer: %v", err)
	}
	if err := p.notifWriter.Close(); err != nil {
		log.Printf("Error closing Notif writer: %v", err)
	}
}
