package adapters

import "context"

// UserResolver fetches user contact details for notification delivery
// Implementations should connect to a User Service or similar
type UserResolver interface {
	// ResolveRecipient fetches contact details for a user
	// Returns nil if user not found or has no valid contact methods
	ResolveRecipient(ctx context.Context, userID string) (*RecipientInfo, error)
}

// RecipientInfo contains contact details for notification delivery
type RecipientInfo struct {
	UserID       string
	Email        string
	PhoneNumber  string
	DeviceTokens []DeviceToken   // Uses DeviceToken from user_preference.go
	Preferences  map[string]bool // channel -> enabled
}

// DefaultUserResolver returns recipient info from notification metadata
// This is used when no external User Service is configured
type DefaultUserResolver struct{}

func (r *DefaultUserResolver) ResolveRecipient(ctx context.Context, userID string) (*RecipientInfo, error) {
	// Default resolver returns minimal info - channels must have recipient info in notification data
	return &RecipientInfo{
		UserID: userID,
	}, nil
}
