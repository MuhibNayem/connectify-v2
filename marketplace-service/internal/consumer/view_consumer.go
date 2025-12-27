package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/shared-entity/pkg/batch"
	"github.com/segmentio/kafka-go"
)

type ViewConsumer struct {
	reader    *kafka.Reader
	repo      *repository.MarketplaceRepository
	logger    *slog.Logger
	processor *batch.Processor[kafka.Message]
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewViewConsumer(brokers []string, repo *repository.MarketplaceRepository, logger *slog.Logger) *ViewConsumer {
	if logger == nil {
		logger = slog.Default()
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  "marketplace-view-counter-group",
		Topic:    "marketplace-product-views",
		MinBytes: 10e3,            // 10KB
		MaxBytes: 10e6,            // 10MB
		MaxWait:  1 * time.Second, // Wait for batch from Kafka
	})

	ctx, cancel := context.WithCancel(context.Background())

	vc := &ViewConsumer{
		reader: reader,
		repo:   repo,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize Batch Processor
	vc.processor = batch.NewProcessor(batch.Config{
		BatchSize:     1000,
		FlushInterval: 5 * time.Second,
		Logger:        logger,
	}, vc.processBatch)

	return vc
}

func (c *ViewConsumer) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				msg, err := c.reader.FetchMessage(c.ctx)
				if err != nil {
					if c.ctx.Err() != nil {
						return // shutting down
					}
					c.logger.Error("Failed to fetch message", "error", err)
					time.Sleep(1 * time.Second)
					continue
				}

				if err := c.processor.Add(c.ctx, msg); err != nil {
					c.logger.Error("Failed to add message to batch processor", "error", err)
				}
			}
		}
	}()
	c.logger.Info("ViewConsumer started")
}

func (c *ViewConsumer) processBatch(ctx context.Context, messages []kafka.Message) error {
	if len(messages) == 0 {
		return nil
	}

	// Aggregate Counts
	viewCounts := make(map[string]int64)
	for _, msg := range messages {
		// Event format: {"product_id": "xyz", ...} or just assume Key is ProductID if structured that way.
		// Let's assume Payload is JSON Event.
		var event struct {
			ProductID string `json:"product_id"`
		}
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Warn("Failed to unmarshal view event", "error", err, "value", string(msg.Value))
			continue
		}
		if event.ProductID != "" {
			viewCounts[event.ProductID]++
		}
	}

	// Bulk Update DB
	if err := c.repo.BatchIncrementViews(ctx, viewCounts); err != nil {
		// Retry?
		// For view counts, it's often acceptable to drop on fatal DB errors rather than block forever.
		// But log strictly.
		c.logger.Error("Failed to bulk update view counts", "error", err)
		return err
	}

	// Commit Offsets (only after successful DB update)
	if err := c.reader.CommitMessages(ctx, messages...); err != nil {
		c.logger.Error("Failed to commit messages", "error", err)
		return err
	}

	c.logger.Info("Processed view batch", "event_count", len(messages), "unique_products", len(viewCounts))
	return nil
}

func (c *ViewConsumer) Stop() {
	c.cancel()
	c.wg.Wait()
	c.processor.Stop()
	if err := c.reader.Close(); err != nil {
		c.logger.Error("Failed to close Kafka reader", "error", err)
	}
	c.logger.Info("ViewConsumer stopped")
}
