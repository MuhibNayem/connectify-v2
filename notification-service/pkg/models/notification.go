package models

import "time"

// Notification represents a generic notification event.
// This model is intentionally generic to work across all industries.
type Notification struct {
	// ID is the unique identifier for this notification
	ID string `json:"id" bson:"_id,omitempty"`

	// RecipientID identifies who should receive this notification
	RecipientID string `json:"recipient_id" bson:"recipient_id"`

	// SenderID identifies who triggered this notification (optional)
	SenderID string `json:"sender_id,omitempty" bson:"sender_id,omitempty"`

	// Type is the notification type (e.g., "order_shipped", "friend_request", "payment_received")
	// YOU define your own types based on your industry/use case
	Type string `json:"type" bson:"type"`

	// Title is a short, human-readable title
	Title string `json:"title" bson:"title"`

	// Body is the main notification message
	Body string `json:"body" bson:"body"`

	// ImageURL is an optional image to display with the notification
	ImageURL string `json:"image_url,omitempty" bson:"image_url,omitempty"`

	// ActionURL is where to navigate when the notification is clicked
	ActionURL string `json:"action_url,omitempty" bson:"action_url,omitempty"`

	// Priority controls delivery priority (normal, high, critical)
	Priority Priority `json:"priority" bson:"priority"`

	// Channels specifies which channels to use (if empty, uses all enabled channels)
	Channels []string `json:"channels,omitempty" bson:"channels,omitempty"`

	// Data contains arbitrary metadata for this notification
	// Use this for industry-specific fields that your app needs
	Data map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`

	// Read indicates if the recipient has read this notification
	Read bool `json:"read" bson:"read"`

	// ReadAt is when the notification was read
	ReadAt *time.Time `json:"read_at,omitempty" bson:"read_at,omitempty"`

	// DeliveredAt tracks when the notification was delivered to each channel
	DeliveredAt map[string]time.Time `json:"delivered_at,omitempty" bson:"delivered_at,omitempty"`

	// FailedChannels tracks channels that failed to deliver
	FailedChannels []string `json:"failed_channels,omitempty" bson:"failed_channels,omitempty"`

	// TemplateID for template-based notifications (optional)
	TemplateID string `json:"template_id,omitempty" bson:"template_id,omitempty"`

	// TemplateData for rendering templates
	TemplateData map[string]interface{} `json:"template_data,omitempty" bson:"template_data,omitempty"`

	// TTL is how long to keep this notification before auto-deletion
	TTL *time.Duration `json:"ttl,omitempty" bson:"ttl,omitempty"`

	// ExpiresAt is when this notification expires
	ExpiresAt *time.Time `json:"expires_at,omitempty" bson:"expires_at,omitempty"`

	// TenantID for multi-tenant systems
	TenantID string `json:"tenant_id,omitempty" bson:"tenant_id,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// Priority levels for notifications
type Priority string

const (
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical" // Bypasses rate limiting and quiet hours
)

// CreateNotificationRequest is the request to create a new notification
type CreateNotificationRequest struct {
	RecipientID    string                 `json:"recipient_id" validate:"required"`
	SenderID       string                 `json:"sender_id,omitempty"`
	Type           string                 `json:"type" validate:"required"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	ImageURL       string                 `json:"image_url,omitempty"`
	ActionURL      string                 `json:"action_url,omitempty"`
	Priority       Priority               `json:"priority,omitempty"`
	Channels       []string               `json:"channels,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	TemplateID     string                 `json:"template_id,omitempty"`
	TemplateData   map[string]interface{} `json:"template_data,omitempty"`
	TTL            *time.Duration         `json:"ttl,omitempty"`
	TenantID       string                 `json:"tenant_id,omitempty"`
	IdempotencyKey string                 `json:"-"` // Header: Idempotency-Key
}

// ListNotificationsRequest is the request to list notifications
type ListNotificationsRequest struct {
	RecipientID string     `json:"recipient_id" validate:"required"`
	Types       []string   `json:"types,omitempty"`    // Filter by types
	Read        *bool      `json:"read,omitempty"`     // Filter by read status
	Channels    []string   `json:"channels,omitempty"` // Filter by channels
	Priority    Priority   `json:"priority,omitempty"` // Filter by priority
	Since       *time.Time `json:"since,omitempty"`    // Filter by created_at >= since
	Until       *time.Time `json:"until,omitempty"`    // Filter by created_at <= until
	Cursor      string     `json:"cursor,omitempty"`   // Cursor for pagination
	Limit       int        `json:"limit,omitempty"`    // Page size (default: 20, max: 100)
	Sort        string     `json:"sort,omitempty"`     // Sort field (e.g., "created_at", "-created_at")
	TenantID    string     `json:"tenant_id,omitempty"`
}

// ListNotificationsResponse is the response for listing notifications
type ListNotificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int64          `json:"total"`
	NextCursor    string         `json:"next_cursor,omitempty"`
	HasMore       bool           `json:"has_more"`
}

// BatchMarkAsReadRequest is the request to mark multiple notifications as read
type BatchMarkAsReadRequest struct {
	RecipientID     string     `json:"recipient_id" validate:"required"`
	NotificationIDs []string   `json:"notification_ids,omitempty"` // Specific IDs
	MarkAllAsRead   bool       `json:"mark_all_as_read,omitempty"` // Mark all as read
	Types           []string   `json:"types,omitempty"`            // Filter by types
	Before          *time.Time `json:"before,omitempty"`           // Mark all before this date
	TenantID        string     `json:"tenant_id,omitempty"`
}
