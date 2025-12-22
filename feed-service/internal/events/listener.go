package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"gitlab.com/spydotech-group/feed-service/internal/config"
	"gitlab.com/spydotech-group/feed-service/internal/repository"
	"gitlab.com/spydotech-group/shared-entity/events"
)

type EventListener struct {
	cfg       *config.Config
	repo      *repository.FeedRepository
	graphRepo *repository.GraphRepository
	readers   []*kafka.Reader
}

func NewEventListener(cfg *config.Config, repo *repository.FeedRepository, graphRepo *repository.GraphRepository) *EventListener {
	return &EventListener{
		cfg:       cfg,
		repo:      repo,
		graphRepo: graphRepo,
		readers:   []*kafka.Reader{},
	}
}

func (l *EventListener) Start(ctx context.Context) {
	// User Events Reader
	l.startReader(ctx, "user-events", "feed-service-users", l.handleUserEvent)

	// Friendship Events Reader (Assuming topic name "friendship-events" based on analysis)
	// Note: previous analysis showed topic might be generic or specific.
	// Usually topic is set in config. Assuming "friendship-events" for now.
	// If not, we might need to verify the topic name in messaging-app config.
	l.startReader(ctx, "friendship-events", "feed-service-friendships", l.handleFriendshipEvent)
}

func (l *EventListener) startReader(ctx context.Context, topic, groupID string, handler func(context.Context, []byte) error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  l.cfg.KafkaBrokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	l.readers = append(l.readers, reader)

	go func() {
		defer func() {
			if err := reader.Close(); err != nil {
				log.Printf("Error closing reader for %s: %v", topic, err)
			}
		}()
		log.Printf("Started Kafka consumer for topic: %s", topic)
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				// Check for context cancellation or closing
				if ctx.Err() != nil {
					return
				}
				// Log but don't crash on temporary read errors
				log.Printf("Error reading message from %s: %v", topic, err)
				time.Sleep(1 * time.Second)
				continue
			}

			if err := handler(ctx, m.Value); err != nil {
				log.Printf("Error handling message from %s: %v", topic, err)
			}
		}
	}()
}

func (l *EventListener) handleUserEvent(ctx context.Context, value []byte) error {
	// UserUpdatedEvent structure from shared-entity or manually defined if sharing is restricted
	// Using shared-entity/events which we saw referenced in user_service.go
	var event events.UserUpdatedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return err
	}

	// Sync to MongoDB Replica (for Profile details)
	if err := l.repo.UpsertUserReplica(ctx, &event); err != nil {
		log.Printf("Error updating user replica: %v", err)
	}
	// Sync to Neo4j (for Graph structure)
	if err := l.graphRepo.SyncUser(ctx, event.UserID); err != nil {
		log.Printf("Error syncing user to Neo4j: %v", err)
	}
	return nil // Return nil if both operations are attempted, logging errors internally
}

func (l *EventListener) handleFriendshipEvent(ctx context.Context, value []byte) error {
	var event events.FriendshipEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return err
	}

	// Sync to MongoDB Replica (Legacy, can be removed if fully switched)
	if err := l.repo.UpdateFriendshipReplica(ctx, &event); err != nil {
		log.Printf("Error updating friendship replica: %v", err)
	}
	// Sync to Neo4j (Primary Friend Source)
	if err := l.graphRepo.UpdateFriendship(ctx, event.RequesterID, event.ReceiverID, event.Status); err != nil {
		log.Printf("Error updating friendship in Neo4j: %v", err)
	}
	return nil // Return nil if both operations are attempted, logging errors internally
}

func (l *EventListener) Close() {
	for _, r := range l.readers {
		if err := r.Close(); err != nil {
			log.Printf("Error closing Kafka reader: %v", err)
		}
	}
}
