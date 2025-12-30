package push

import (
	"context"
	"errors"
	"testing"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/channels"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

type mockPushProvider struct {
	errByToken map[string]error
}

func (m *mockPushProvider) SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error {
	if err, ok := m.errByToken[token]; ok {
		return err
	}
	return nil
}

func (m *mockPushProvider) Name() string {
	return "mock"
}

func TestPushChannelSendReturnsCombinedErrors(t *testing.T) {
	provider := &mockPushProvider{
		errByToken: map[string]error{
			"token-1": errors.New("network timeout"),
		},
	}
	channel := NewPushChannel(provider)

	err := channel.Send(context.Background(), &models.Notification{
		Title: "hello",
		Body:  "world",
	}, &channels.ChannelRecipient{
		DeviceTokens: []channels.DeviceToken{
			{Token: "token-1", Platform: "ios"},
			{Token: "token-2", Platform: "android"},
		},
	})

	if err == nil {
		t.Fatalf("expected error from push delivery")
	}

	if !errors.Is(err, provider.errByToken["token-1"]) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
}
