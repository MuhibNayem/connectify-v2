package dlq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"go.uber.org/zap"
)

// FailedEvent represents an event that failed processing
type FailedEvent struct {
	OriginalEvent *adapters.NotificationEvent `json:"original_event"`
	Error         string                      `json:"error"`
	Attempts      int                         `json:"attempts"`
	FailedAt      time.Time                   `json:"failed_at"`
	LastChannel   string                      `json:"last_channel,omitempty"`
}

// Handler manages dead letter queue operations
type Handler struct {
	storage adapters.StorageAdapter
	logger  *zap.Logger
}

// NewHandler creates a new DLQ handler
func NewHandler(storage adapters.StorageAdapter, logger *zap.Logger) *Handler {
	return &Handler{
		storage: storage,
		logger:  logger,
	}
}

// Send sends a failed event to the dead letter queue
func (h *Handler) Send(ctx context.Context, event *adapters.NotificationEvent, err error, attempts int) error {
	failedEvent := &FailedEvent{
		OriginalEvent: event,
		Error:         err.Error(),
		Attempts:      attempts,
		FailedAt:      time.Now(),
	}

	// In production, this would publish to a separate Kafka topic "notifications.dlq"
	// For now, we persist to storage with a special type
	data, _ := json.Marshal(failedEvent)

	h.logger.Error("💀 Event sent to DLQ",
		zap.String("event_id", event.ID),
		zap.String("error", err.Error()),
		zap.Int("attempts", attempts),
		zap.ByteString("payload", data))

	// Store in a DLQ collection/table for later inspection and reprocessing
	dlqNotification := &adapters.Notification{
		ID:             "dlq_" + event.ID,
		Type:           "DLQ_FAILED",
		Data:           map[string]interface{}{"failed_event": string(data)},
		Priority:       "LOW",
		FailedChannels: []string{failedEvent.LastChannel},
	}

	return h.storage.Create(ctx, dlqNotification)
}

// Reprocess attempts to reprocess events from the DLQ
func (h *Handler) Reprocess(ctx context.Context, eventID string) error {
	// Fetch from DLQ storage
	// Re-publish to main queue
	// This would be called by an admin API or scheduled job
	h.logger.Info("🔄 Reprocessing DLQ event", zap.String("event_id", eventID))
	return nil
}

// ListFailed returns all events in the DLQ for inspection
func (h *Handler) ListFailed(ctx context.Context, limit int) ([]*FailedEvent, error) {
	// Query storage for DLQ entries
	// This enables admin dashboards to view failed events
	return nil, nil
}
