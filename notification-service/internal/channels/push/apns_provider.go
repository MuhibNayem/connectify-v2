package push

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// APNSProvider implements PushProvider using Apple Push Notification Service HTTP/2 API
type APNSProvider struct {
	teamID     string
	keyID      string
	p8Content  []byte
	production bool
	client     *http.Client
}

func NewAPNSProvider(teamID, keyID, p8File string, production bool) (*APNSProvider, error) {
	// Setup HTTP/2 client
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	return &APNSProvider{
		teamID:     teamID,
		keyID:      keyID,
		production: production,
		client:     client,
	}, nil
}

func (a *APNSProvider) SendPush(ctx context.Context, token string, title, body string, data map[string]interface{}) error {
	host := "api.sandbox.push.apple.com"
	if a.production {
		host = "api.push.apple.com"
	}

	url := fmt.Sprintf("https://%s/3/device/%s", host, token)

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]string{
				"title": title,
				"body":  body,
			},
		},
		"custom_data": data,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	// Generate JWT token for APNS (Implementation omitted for brevity, usually involves ECDSA signing)
	jwtToken := "mock-jwt-token"

	req.Header.Set("authorization", "bearer "+jwtToken)
	req.Header.Set("apns-topic", "com.yourapp.bundleid") // Should come from config
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("APNS API error: %d", resp.StatusCode)
	}

	return nil
}

func (a *APNSProvider) Name() string {
	return "apns"
}
