package adapters

import "context"

// QueueAdapter defines the interface for message queue integration.
// Implement this interface to use any queue (Kafka, RabbitMQ, Redis Streams, etc.)
type QueueAdapter interface {
	// Publish publishes a notification event to the queue
	Publish(ctx context.Context, event *NotificationEvent) error

	// PublishBatch publishes multiple events in a batch
	PublishBatch(ctx context.Context, events []*NotificationEvent) error

	// Subscribe starts consuming events from the queue
	Subscribe(ctx context.Context, handler EventHandler) error

	// Close closes the queue connection
	Close() error

	// HealthCheck verifies queue connectivity
	HealthCheck(ctx context.Context) error
}

// NotificationEvent represents an event in the queue
type NotificationEvent struct {
	ID        string
	Type      string
	Payload   map[string]interface{}
	Timestamp string
	TenantID  string
}

// EventHandler processes incoming events
type EventHandler func(ctx context.Context, event *NotificationEvent) error
