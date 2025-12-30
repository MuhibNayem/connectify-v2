package sms

import (
	"context"
	"fmt"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/channels"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

// SMSProvider defines the interface for ANY SMS provider (Twilio, AWS SNS, GCP, etc.)
type SMSProvider interface {
	SendSMS(ctx context.Context, to, body string) error
	Name() string
}

// SMSChannel is the generic channel that uses a pluggable provider
type SMSChannel struct {
	provider SMSProvider
	enabled  bool
}

func NewSMSChannel(provider SMSProvider) *SMSChannel {
	return &SMSChannel{
		provider: provider,
		enabled:  provider != nil,
	}
}

func (s *SMSChannel) Name() string {
	return "sms"
}

func (s *SMSChannel) Send(ctx context.Context, notification *models.Notification, recipient *channels.ChannelRecipient) error {
	if !s.enabled || recipient.PhoneNumber == "" {
		return fmt.Errorf("sms channel disabled or missing phone number")
	}

	return s.provider.SendSMS(ctx, recipient.PhoneNumber, notification.Body)
}

func (s *SMSChannel) SendBatch(ctx context.Context, notifications []*models.Notification, recipients []*channels.ChannelRecipient) error {
	for i, notif := range notifications {
		if err := s.Send(ctx, notif, recipients[i]); err != nil {
			// Log error but continue
			continue
		}
	}
	return nil
}

func (s *SMSChannel) IsEnabled() bool {
	return s.enabled
}

func (s *SMSChannel) HealthCheck(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	return nil
}
