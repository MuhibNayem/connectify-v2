package producer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.com/spydotech-group/shared-entity/models"
)

// StoryBroadcaster interface for real-time event publishing
type StoryBroadcaster interface {
	PublishStoryCreated(ctx context.Context, event StoryCreatedEvent)
	PublishStoryDeleted(ctx context.Context, event StoryDeletedEvent)
	PublishStoryViewed(ctx context.Context, event StoryViewedEvent)
	PublishStoryReaction(ctx context.Context, event StoryReactionEvent)
	Close() error
}

// Event types
type StoryCreatedEvent struct {
	StoryID   string            `json:"story_id"`
	UserID    string            `json:"user_id"`
	Author    models.PostAuthor `json:"author"`
	MediaURL  string            `json:"media_url"`
	MediaType string            `json:"media_type"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type StoryDeletedEvent struct {
	StoryID string `json:"story_id"`
	UserID  string `json:"user_id"`
}

type StoryViewedEvent struct {
	StoryID  string    `json:"story_id"`
	OwnerID  string    `json:"owner_id"` // Story owner to be notified
	ViewerID string    `json:"viewer_id"`
	ViewedAt time.Time `json:"viewed_at"`
}

type StoryReactionEvent struct {
	StoryID      string    `json:"story_id"`
	UserID       string    `json:"user_id"`
	ReactionType string    `json:"reaction_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// StoryProducer implements StoryBroadcaster
type StoryProducer struct {
	writer *kafka.Writer
}

func NewStoryProducer(brokers []string, topic string) *StoryProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		Async:        true,
	}

	return &StoryProducer{writer: writer}
}

func (p *StoryProducer) publish(ctx context.Context, eventType string, payload interface{}) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal story event payload: %v", err)
		return
	}

	wsEvent := models.WebSocketEvent{
		Type: eventType,
		Data: payloadData,
	}

	data, err := json.Marshal(wsEvent)
	if err != nil {
		log.Printf("Failed to marshal story event: %v", err)
		return
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(eventType),
		Value: data,
	})
	if err != nil {
		log.Printf("Failed to publish story event: %v", err)
	}
}

func (p *StoryProducer) PublishStoryCreated(ctx context.Context, event StoryCreatedEvent) {
	p.publish(ctx, "STORY_CREATED", event)
}

func (p *StoryProducer) PublishStoryDeleted(ctx context.Context, event StoryDeletedEvent) {
	p.publish(ctx, "STORY_DELETED", event)
}

func (p *StoryProducer) PublishStoryViewed(ctx context.Context, event StoryViewedEvent) {
	p.publish(ctx, "STORY_VIEWED", event)
}

func (p *StoryProducer) PublishStoryReaction(ctx context.Context, event StoryReactionEvent) {
	p.publish(ctx, "STORY_REACTION", event)
}

func (p *StoryProducer) Close() error {
	return p.writer.Close()
}
