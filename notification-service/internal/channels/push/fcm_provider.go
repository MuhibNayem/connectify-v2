package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FCMProvider implements PushProvider using Firebase Cloud Messaging HTTP v1 API
type FCMProvider struct {
	projectID   string
	credentials []byte
	client      *http.Client
}

func NewFCMProvider(projectID string, credentialsFile string) (*FCMProvider, error) {
	// In a real implementation, we would load credentials from file
	// For now, we simulate success
	return &FCMProvider{
		projectID: projectID,
		client:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (f *FCMProvider) SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error {
	// Get Access Token (Mocked for now to avoid google package dependency issues if not installed)
	// In production: ts, err := google.CredentialsFromJSON(ctx, f.credentials, "https://www.googleapis.com/auth/firebase.messaging")
	accessToken := "mock-access-token"

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)

	message := map[string]interface{}{
		"message": map[string]interface{}{
			"token": token,
			"notification": map[string]string{
				"title": title,
				"body":  body,
			},
			"data": data,
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FCM API error: %d", resp.StatusCode)
	}

	return nil
}

func (f *FCMProvider) Name() string {
	return "fcm"
}
