package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// FCMProvider implements PushProvider using Firebase Cloud Messaging HTTP v1 API
type FCMProvider struct {
	projectID   string
	tokenSource oauth2.TokenSource
	client      *http.Client
}

func NewFCMProvider(projectID string, credentialsJSON []byte) (*FCMProvider, error) {
	// Create TokenSource from service account JSON
	creds, err := google.CredentialsFromJSON(context.Background(), credentialsJSON, "https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &FCMProvider{
		projectID:   projectID,
		tokenSource: creds.TokenSource,
		client:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (f *FCMProvider) SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error {
	// Get Access Token
	oauthToken, err := f.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get oauth token: %w", err)
	}
	accessToken := oauthToken.AccessToken

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
