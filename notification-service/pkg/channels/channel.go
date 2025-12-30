package channels

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

// Channel defines the interface that all notification channels must implement.
// A channel is responsible for delivering notifications through a specific medium
// (e.g., email, push notification, SMS, webhook, etc.).
type Channel interface {
	// Name returns the unique name of this channel (e.g., "email", "push", "sms")
	Name() string

	// Send delivers a notification through this channel.
	// Returns error if delivery fails.
	Send(ctx context.Context, notification *models.Notification, recipient *ChannelRecipient) error

	// SendBatch delivers multiple notifications in a batch (if supported).
	// Falls back to individual Send calls if batch not supported.
	SendBatch(ctx context.Context, notifications []*models.Notification, recipients []*ChannelRecipient) error

	// IsEnabled checks if this channel is enabled globally
	IsEnabled() bool

	// HealthCheck verifies the channel is operational
	HealthCheck(ctx context.Context) error
}

// ChannelRecipient contains channel-specific recipient information
type ChannelRecipient struct {
	UserID string

	// For email channel
	Email string

	// For SMS channel
	PhoneNumber string

	// For push channel
	DeviceTokens []DeviceToken

	// For webhook channel
	WebhookURL string

	// Custom metadata for custom channels
	Metadata map[string]interface{}
}

// DeviceToken represents a device for push notifications
type DeviceToken struct {
	Token    string
	Platform string // "ios", "android", "web"
}

// ChannelConfig contains configuration for a channel plugin
type ChannelConfig struct {
	Enabled       bool
	WorkerCount   int
	QueueSize     int
	RetryAttempts int
	Timeout       int                    // seconds
	Config        map[string]interface{} // Channel-specific configuration
}
