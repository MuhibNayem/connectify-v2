package adapters

import (
	"context"

	"errors"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

var ErrNotFound = errors.New("not found")

// StorageAdapter defines the interface for notification storage.
// Implement this interface to use any database (MongoDB, PostgreSQL, MySQL, etc.)
type StorageAdapter interface {
	// Create stores a new notification
	Create(ctx context.Context, notification *Notification) error

	// Get retrieves a notification by ID
	Get(ctx context.Context, id string) (*Notification, error)

	// List retrieves notifications with filtering and pagination
	List(ctx context.Context, query *ListQuery) (*ListResult, error)

	// Update updates a notification
	Update(ctx context.Context, id string, updates map[string]interface{}) error

	// Delete deletes a notification
	Delete(ctx context.Context, id string) error

	// GetUnreadCount returns the count of unread notifications
	GetUnreadCount(ctx context.Context, recipientID string) (int64, error)

	// BatchMarkAsRead marks multiple notifications as read
	BatchMarkAsRead(ctx context.Context, recipientID string, ids []string) (int64, error)

	// DeleteExpired deletes expired notifications
	DeleteExpired(ctx context.Context) (int64, error)

	// HealthCheck verifies database connectivity
	HealthCheck(ctx context.Context) error

	// SaveDeliveryState upserts the delivery state for a notification
	SaveDeliveryState(ctx context.Context, state *models.NotificationDeliveryState) error

	// GetDeliveryState retrieves delivery state for a notification
	GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error)

	// --- Transactional Outbox Support ---

	// CreateWithOutbox atomically saves the notification and an outbox event
	CreateWithOutbox(ctx context.Context, notification *Notification, event *NotificationEvent) error

	// CreateBatchWithOutbox atomically saves multiple notifications and outbox events
	CreateBatchWithOutbox(ctx context.Context, notifications []*Notification, events []*NotificationEvent) error

	// GetPendingOutboxEvents retrieves events that haven't been published yet
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]*NotificationEvent, error)

	// DeleteOutboxEvent removes an event from the outbox (after successful publish)
	DeleteOutboxEvent(ctx context.Context, eventID string) error
}

// Notification is the storage model
type Notification struct {
	ID             string
	RecipientID    string
	SenderID       string
	Type           string
	Title          string
	Body           string
	ImageURL       string
	ActionURL      string
	Priority       string
	Channels       []string
	Data           map[string]interface{}
	Read           bool
	ReadAt         *string
	DeliveredAt    map[string]string
	FailedChannels []string
	TemplateID     string
	TemplateData   map[string]interface{}
	ExpiresAt      *string
	TenantID       string
	CreatedAt      string
	UpdatedAt      string
}

// ListQuery defines query parameters for listing notifications
type ListQuery struct {
	RecipientID string
	Types       []string
	Read        *bool
	Channels    []string
	Priority    string
	Since       *string
	Until       *string
	Cursor      string
	Limit       int
	Sort        string
	TenantID    string
}

// ListResult contains paginated results
type ListResult struct {
	Notifications []*Notification
	Total         int64
	NextCursor    string
	HasMore       bool
}
