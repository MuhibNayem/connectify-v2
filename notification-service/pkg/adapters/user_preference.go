package adapters

import "context"

// UserPreferenceAdapter defines the interface for fetching user notification preferences.
// Implement this interface to integrate with your existing user management system.
//
// Example implementations:
//   - HTTP API: Call your user service's REST endpoint
//   - gRPC: Call your user service's gRPC method
//   - Database: Query directly from your user database
//   - Static: Return hardcoded preferences for testing
type UserPreferenceAdapter interface {
	// GetPreferences retrieves notification preferences for a user.
	// Returns nil preferences if user not found or uses defaults.
	GetPreferences(ctx context.Context, userID string) (*UserPreferences, error)

	// UpdatePreferences updates notification preferences for a user.
	// Returns updated preferences on success.
	UpdatePreferences(ctx context.Context, userID string, prefs *UserPreferences) error
}

// UserPreferences represents user notification preferences.
// All fields are optional - nil means "use system default".
type UserPreferences struct {
	// UserID is the unique identifier for the user
	UserID string `json:"user_id"`

	// Global channel preferences (nil = enabled by default)
	InAppEnabled   *bool `json:"inapp_enabled,omitempty"`
	PushEnabled    *bool `json:"push_enabled,omitempty"`
	EmailEnabled   *bool `json:"email_enabled,omitempty"`
	SMSEnabled     *bool `json:"sms_enabled,omitempty"`
	WebhookEnabled *bool `json:"webhook_enabled,omitempty"`

	// Per-type channel preferences
	// Key: notification type (e.g., "order_shipped", "friend_request")
	// Value: channels enabled for this type
	TypePreferences map[string]*ChannelPreferences `json:"type_preferences,omitempty"`

	// Quiet hours configuration
	QuietHours *QuietHours `json:"quiet_hours,omitempty"`

	// Contact information (required for email/sms channels)
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`

	// Device tokens for push notifications
	DeviceTokens []DeviceToken `json:"device_tokens,omitempty"`

	// Webhook URL for webhook channel
	WebhookURL string `json:"webhook_url,omitempty"`

	// Locale for template rendering (e.g., "en-US", "es-ES")
	Locale string `json:"locale,omitempty"`

	// Timezone for quiet hours (e.g., "America/New_York")
	Timezone string `json:"timezone,omitempty"`

	// Custom metadata - use this for industry-specific fields
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChannelPreferences defines which channels are enabled for a notification type
type ChannelPreferences struct {
	InApp   bool `json:"inapp"`
	Push    bool `json:"push"`
	Email   bool `json:"email"`
	SMS     bool `json:"sms"`
	Webhook bool `json:"webhook"`
}

// QuietHours defines a time period when notifications should not be sent
type QuietHours struct {
	Enabled    bool   `json:"enabled"`
	StartTime  string `json:"start_time"`   // HH:MM format, e.g., "22:00"
	EndTime    string `json:"end_time"`     // HH:MM format, e.g., "08:00"
	Timezone   string `json:"timezone"`     // IANA timezone, e.g., "America/New_York"
	DaysOfWeek []int  `json:"days_of_week"` // 0=Sunday, 6=Saturday
}

// DeviceToken represents a device registered for push notifications
type DeviceToken struct {
	Token    string `json:"token"`
	Platform string `json:"platform"` // "ios", "android", "web"
	Enabled  bool   `json:"enabled"`
}
