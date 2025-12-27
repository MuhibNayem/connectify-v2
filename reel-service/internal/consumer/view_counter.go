package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/reel-service/internal/producer"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ViewCountConsumer struct {
	reader     *kafka.Reader
	collection *mongo.Collection
	logger     *slog.Logger

	batchSize     int
	flushInterval time.Duration

	mu         sync.Mutex
	viewCounts map[string]int64
	stopCh     chan struct{}
	doneCh     chan struct{}
}

type ViewCountConsumerConfig struct {
	Brokers       []string
	Topic         string
	GroupID       string
	BatchSize     int
	FlushInterval time.Duration
}

func DefaultViewCountConfig(brokers []string, topic string) ViewCountConsumerConfig {
	return ViewCountConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       "reel-view-counter",
		BatchSize:     500,
		FlushInterval: 5 * time.Second,
	}
}

func NewViewCountConsumer(cfg ViewCountConsumerConfig, collection *mongo.Collection, logger *slog.Logger) *ViewCountConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	return &ViewCountConsumer{
		reader:        reader,
		collection:    collection,
		logger:        logger,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		viewCounts:    make(map[string]int64),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

func (c *ViewCountConsumer) Start(ctx context.Context) {
	go c.consume(ctx)
	go c.periodicFlush(ctx)
}

func (c *ViewCountConsumer) consume(ctx context.Context) {
	defer close(c.doneCh)

	for {
		select {
		case <-ctx.Done():
			c.flush(context.Background())
			return
		case <-c.stopCh:
			c.flush(context.Background())
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.Error("Failed to read message", "error", err)
				continue
			}

			if string(msg.Key) == "reel.viewed" {
				var event producer.ReelViewedEvent
				if err := json.Unmarshal(msg.Value, &event); err != nil {
					c.logger.Error("Failed to unmarshal view event", "error", err)
					continue
				}

				c.accumulate(event.ReelID)
			}
		}
	}
}

func (c *ViewCountConsumer) accumulate(reelID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viewCounts[reelID]++

	totalViews := int64(0)
	for _, count := range c.viewCounts {
		totalViews += count
	}

	if totalViews >= int64(c.batchSize) {
		go c.flush(context.Background())
	}
}

func (c *ViewCountConsumer) periodicFlush(ctx context.Context) {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.flush(ctx)
		}
	}
}

func (c *ViewCountConsumer) flush(ctx context.Context) {
	c.mu.Lock()
	if len(c.viewCounts) == 0 {
		c.mu.Unlock()
		return
	}

	batch := c.viewCounts
	c.viewCounts = make(map[string]int64)
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	totalUpdated := int64(0)
	for reelID, count := range batch {
		oid, err := primitive.ObjectIDFromHex(reelID)
		if err != nil {
			c.logger.Error("Invalid reel ID", "reel_id", reelID, "error", err)
			continue
		}

		result, err := c.collection.UpdateOne(
			ctx,
			bson.M{"_id": oid},
			bson.M{"$inc": bson.M{"views": count}},
		)
		if err != nil {
			c.logger.Error("Failed to update view count", "reel_id", reelID, "error", err)
			c.mu.Lock()
			c.viewCounts[reelID] += count
			c.mu.Unlock()
			continue
		}

		if result.ModifiedCount > 0 {
			totalUpdated += count
		}
	}

	if totalUpdated > 0 {
		c.logger.Info("Flushed view counts to MongoDB",
			"reels_updated", len(batch),
			"total_views", totalUpdated,
		)
	}
}

func (c *ViewCountConsumer) Stop() error {
	close(c.stopCh)
	<-c.doneCh
	return c.reader.Close()
}
