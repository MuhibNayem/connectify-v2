package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/kafka"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/repository"
	kafkago "github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Processor processes outbox events asynchronously
type Processor struct {
	repo        *Repository
	graphClient repository.GraphClient
	producer    *kafka.MessageProducer
	logger      *slog.Logger
	batchSize   int64
	interval    time.Duration
	stopChan    chan struct{}
}

// NewProcessor creates a new outbox processor
func NewProcessor(
	repo *Repository,
	graphClient repository.GraphClient,
	producer *kafka.MessageProducer,
	logger *slog.Logger,
) *Processor {
	return &Processor{
		repo:        repo,
		graphClient: graphClient,
		producer:    producer,
		logger:      logger,
		batchSize:   100,
		interval:    1 * time.Second,
		stopChan:    make(chan struct{}),
	}
}

// Start begins processing outbox events in background
func (p *Processor) Start(ctx context.Context) {
	p.logger.Info("Starting outbox processor")

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Outbox processor stopping (context cancelled)")
			return
		case <-p.stopChan:
			p.logger.Info("Outbox processor stopping (stop signal)")
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

// Stop signals the processor to stop
func (p *Processor) Stop() {
	close(p.stopChan)
}

func (p *Processor) processBatch(ctx context.Context) {
	events, err := p.repo.GetPending(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("Failed to fetch pending outbox events", "error", err)
		return
	}

	if len(events) == 0 {
		return
	}

	p.logger.Debug("Processing outbox events", "count", len(events))

	for _, event := range events {
		if err := p.processEvent(ctx, event); err != nil {
			p.logger.Warn("Failed to process outbox event",
				"event_id", event.ID.Hex(),
				"event_type", event.EventType,
				"retry_count", event.RetryCount,
				"error", err,
			)
			_ = p.repo.MarkFailed(ctx, event.ID, err.Error(), event.RetryCount, event.MaxRetries)
			continue
		}

		if err := p.repo.MarkProcessed(ctx, event.ID); err != nil {
			p.logger.Error("Failed to mark event as processed", "event_id", event.ID.Hex(), "error", err)
		} else {
			p.logger.Debug("Processed outbox event", "event_id", event.ID.Hex(), "event_type", event.EventType)
		}
	}
}

func (p *Processor) processEvent(ctx context.Context, event *Event) error {
	// Step 1: Update graph database (if graph client available)
	if p.graphClient != nil {
		if err := p.syncToGraph(ctx, event); err != nil {
			return err
		}
	}

	// Step 2: Publish to Kafka
	if p.producer != nil {
		if err := p.publishToKafka(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (p *Processor) syncToGraph(ctx context.Context, event *Event) error {
	requesterID, _ := primitive.ObjectIDFromHex(event.Payload["requester_id"].(string))
	receiverID, _ := primitive.ObjectIDFromHex(event.Payload["receiver_id"].(string))

	switch event.EventType {
	case EventFriendRequestSent:
		return p.graphClient.SendRequest(ctx, requesterID, receiverID)

	case EventFriendRequestAccepted:
		return p.graphClient.AcceptRequest(ctx, requesterID, receiverID)

	case EventFriendRequestRejected:
		return p.graphClient.RejectRequest(ctx, requesterID, receiverID)

	case EventUnfriended:
		return p.graphClient.Unfriend(ctx, requesterID, receiverID)

	case EventUserBlocked:
		return p.graphClient.Block(ctx, requesterID, receiverID)

	case EventUserUnblocked:
		return p.graphClient.Unblock(ctx, requesterID, receiverID)

	default:
		p.logger.Warn("Unknown event type for graph sync", "event_type", event.EventType)
		return nil
	}
}

func (p *Processor) publishToKafka(ctx context.Context, event *Event) error {
	// Convert outbox event to Kafka message
	kafkaEvent := map[string]interface{}{
		"event_id":   event.ID.Hex(),
		"event_type": event.EventType,
		"payload":    event.Payload,
		"timestamp":  event.CreatedAt,
	}

	value, err := json.Marshal(kafkaEvent)
	if err != nil {
		return err
	}

	msg := kafkago.Message{
		Key:   []byte(event.AggregateID.Hex()),
		Value: value,
	}

	return p.producer.ProduceMessage(ctx, msg)
}
