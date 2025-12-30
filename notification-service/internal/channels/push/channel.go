package push

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/channels"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

// PushProvider defines the interface for ANY push provider (FCM, APNS, OneSignal, etc.)
type PushProvider interface {
	SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error
	Name() string
}

// PushChannel is the generic channel that uses a pluggable provider
type PushChannel struct {
	provider PushProvider
	enabled  bool
}

func NewPushChannel(provider PushProvider) *PushChannel {
	return &PushChannel{
		provider: provider,
		enabled:  provider != nil,
	}
}

func (p *PushChannel) Name() string {
	return "push"
}

func (p *PushChannel) Send(ctx context.Context, notification *models.Notification, recipient *channels.ChannelRecipient) error {
	if !p.enabled || len(recipient.DeviceTokens) == 0 {
		return nil // Skip if disabled or no tokens
	}

	for _, token := range recipient.DeviceTokens {
		// Send to each device
		_ = p.provider.SendPush(ctx, token.Token, notification.Title, notification.Body, notification.Data)
	}

	return nil
}

func (p *PushChannel) SendBatch(ctx context.Context, notifications []*models.Notification, recipients []*channels.ChannelRecipient) error {
	for i, notif := range notifications {
		if err := p.Send(ctx, notif, recipients[i]); err != nil {
			// Log error but continue
			continue
		}
	}
	return nil
}

func (p *PushChannel) IsEnabled() bool {
	return p.enabled
}

func (p *PushChannel) HealthCheck(ctx context.Context) error {
	if !p.enabled {
		return nil
	}
	// Could add provider-specific health check here
	return nil
}
