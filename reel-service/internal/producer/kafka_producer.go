package producer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type ReelBroadcaster interface {
	PublishReelCreated(ctx context.Context, event ReelCreatedEvent)
	PublishReelDeleted(ctx context.Context, event ReelDeletedEvent)
	PublishReelViewed(ctx context.Context, event ReelViewedEvent)
	Close() error
}

type ReelCreatedEvent struct {
	ReelID   string `json:"reel_id"`
	UserID   string `json:"user_id"`
	VideoURL string `json:"video_url"`
}

type ReelDeletedEvent struct {
	ReelID string `json:"reel_id"`
	UserID string `json:"user_id"`
}

type ReelViewedEvent struct {
	ReelID   string `json:"reel_id"`
	ViewerID string `json:"viewer_id"`
}

type ReelProducer struct {
	writer *kafka.Writer
	topic  string
}

func NewReelProducer(brokers []string, topic string) *ReelProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	return &ReelProducer{writer: writer, topic: topic}
}

func (p *ReelProducer) PublishReelCreated(ctx context.Context, event ReelCreatedEvent) {
	p.publish(ctx, "reel.created", event)
}

func (p *ReelProducer) PublishReelDeleted(ctx context.Context, event ReelDeletedEvent) {
	p.publish(ctx, "reel.deleted", event)
}

func (p *ReelProducer) PublishReelViewed(ctx context.Context, event ReelViewedEvent) {
	p.publish(ctx, "reel.viewed", event)
}

func (p *ReelProducer) publish(ctx context.Context, eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal event", "type", eventType, "error", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(eventType),
		Value: data,
	}

	go func() {
		if err := p.writer.WriteMessages(context.Background(), msg); err != nil {
			slog.Error("Failed to publish event", "type", eventType, "error", err)
		}
	}()
}

func (p *ReelProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
