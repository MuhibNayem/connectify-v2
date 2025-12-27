package producer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

type EventProducer struct {
	writer *kafka.Writer
}

func NewEventProducer(brokers []string, topic string) *EventProducer {
	return &EventProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        true,
			Completion: func(messages []kafka.Message, err error) {
				if err != nil {
					log.Printf("Failed to write messages to Kafka: %v", err)
				}
			},
		},
	}
}

func (p *EventProducer) publish(ctx context.Context, eventType string, data interface{}, key string) error {
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

	return p.writer.WriteMessages(ctx, msg)
}

func (p *EventProducer) BroadcastRSVP(event models.EventRSVPEvent) {
	if err := p.publish(context.Background(), "EVENT_RSVP_UPDATE", event, event.EventID); err != nil {
		log.Printf("Error publishing RSVP event: %v", err)
	}
}

func (p *EventProducer) PublishEventUpdated(ctx context.Context, event models.EventUpdatedEvent) {
	if err := p.publish(ctx, "EVENT_UPDATED", event, event.ID); err != nil {
		log.Printf("Error publishing EventUpdated event: %v", err)
	}
}

func (p *EventProducer) PublishEventDeleted(ctx context.Context, event models.EventDeletedEvent) {
	if err := p.publish(ctx, "EVENT_DELETED", event, event.ID); err != nil {
		log.Printf("Error publishing EventDeleted event: %v", err)
	}
}

func (p *EventProducer) PublishPostCreated(ctx context.Context, event models.EventPostCreatedEvent) {
	if err := p.publish(ctx, "EVENT_POST_CREATED", event, event.EventID); err != nil {
		log.Printf("Error publishing PostCreated event: %v", err)
	}
}

func (p *EventProducer) PublishPostReaction(ctx context.Context, event models.EventPostReactionEvent) {
	if err := p.publish(ctx, "EVENT_POST_REACTION", event, event.EventID); err != nil {
		log.Printf("Error publishing PostReaction event: %v", err)
	}
}

func (p *EventProducer) PublishInvitationUpdated(ctx context.Context, event models.EventInvitationUpdatedEvent) {
	if err := p.publish(ctx, "EVENT_INVITATION_UPDATED", event, event.EventID); err != nil {
		log.Printf("Error publishing InvitationUpdated event: %v", err)
	}
}

func (p *EventProducer) PublishCoHostAdded(ctx context.Context, event models.EventCoHostAddedEvent) {
	if err := p.publish(ctx, "EVENT_COHOST_ADDED", event, event.EventID); err != nil {
		log.Printf("Error publishing CoHostAdded event: %v", err)
	}
}

func (p *EventProducer) PublishCoHostRemoved(ctx context.Context, event models.EventCoHostRemovedEvent) {
	if err := p.publish(ctx, "EVENT_COHOST_REMOVED", event, event.EventID); err != nil {
		log.Printf("Error publishing CoHostRemoved event: %v", err)
	}
}

func (p *EventProducer) Close() error {
	return p.writer.Close()
}
