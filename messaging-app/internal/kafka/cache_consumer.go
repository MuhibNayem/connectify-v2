package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type CacheInvalidator struct {
	reader      *kafka.Reader
	redisClient *redis.ClusterClient
}

func NewCacheInvalidator(brokers []string, topic string, groupID string, redisClient *redis.ClusterClient) *CacheInvalidator {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	return &CacheInvalidator{
		reader:      r,
		redisClient: redisClient,
	}
}

func (c *CacheInvalidator) Start(ctx context.Context) {
	go func() {
		log.Printf("Starting Cache Invalidator for topic %s", c.reader.Config().Topic)
		defer c.reader.Close()

		for {
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				// Context canceled or error
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error fetching message in cache invalidator: %v", err)
				time.Sleep(time.Second) // Backoff
				continue
			}

			// Process Message
			if err := c.invalidateHash(ctx, m.Value); err != nil {
				log.Printf("Failed to invalidate cache: %v", err)
			}

			if err := c.reader.CommitMessages(ctx, m); err != nil {
				log.Printf("Error committing message: %v", err)
			}
		}
	}()
}

func (c *CacheInvalidator) invalidateHash(ctx context.Context, value []byte) error {
	var event map[string]interface{}
	if err := json.Unmarshal(value, &event); err != nil {
		return err
	}

	eventType, ok := event["event_type"].(string)
	if !ok {
		// Try fallback structs or just ignore non-standard events
		// user-service emits: "event_type": "UserUpdated", "user_id": ...
		// We only care about UserUpdated for now.
		return nil
	}

	if eventType == "UserUpdated" {
		userID, ok := event["user_id"].(string)
		if ok && userID != "" {
			key := fmt.Sprintf("user:profile:%s", userID)
			return c.redisClient.Del(ctx, key).Err()
		}
	}

	return nil
}

func (c *CacheInvalidator) Close() {
	if c.reader != nil {
		c.reader.Close()
	}
}
