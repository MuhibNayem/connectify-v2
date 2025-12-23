package kafka

import (
	"context"
	"encoding/json"
	"log"
	"messaging-app/internal/websocket"
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"

	"github.com/segmentio/kafka-go"
)

// StoryConsumer consumes story events from Kafka and pushes them to WebSocket clients.
type StoryConsumer struct {
	reader *kafka.Reader
	hub    *websocket.Hub
}

// NewStoryConsumer creates a new StoryConsumer.
func NewStoryConsumer(brokers []string, topic string, groupID string, hub *websocket.Hub) *StoryConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	return &StoryConsumer{
		reader: r,
		hub:    hub,
	}
}

// Start consuming story events from Kafka.
func (c *StoryConsumer) Start(ctx context.Context) {
	log.Printf("Starting Kafka Story Consumer for topic %s", c.reader.Config().Topic)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Kafka Story Consumer for topic %s stopped", c.reader.Config().Topic)
			return
		default:
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				log.Printf("Error fetching story message from Kafka: %v", err)
				continue
			}

			var wsEvent models.WebSocketEvent
			if err := json.Unmarshal(m.Value, &wsEvent); err != nil {
				log.Printf("Error unmarshaling story event: %v, message: %s", err, string(m.Value))
				c.reader.CommitMessages(ctx, m)
				continue
			}

			log.Printf("Received story event: %s", wsEvent.Type)

			// Route story events to FeedEvents channel for processing
			select {
			case c.hub.FeedEvents <- wsEvent:
				// Successfully sent to hub
			default:
				log.Printf("WARNING: FeedEvents channel full. Dropping story event: %s", wsEvent.Type)
			}

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("Error committing story message: %v", err)
			}
		}
	}
}

// Close closes the Kafka reader.
func (c *StoryConsumer) Close() error {
	return c.reader.Close()
}
