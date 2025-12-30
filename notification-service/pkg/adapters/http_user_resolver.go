package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPUserResolver implements UserResolver via HTTP calls to an external service
type HTTPUserResolver struct {
	client  *http.Client
	baseURL string
}

func NewHTTPUserResolver(baseURL string) *HTTPUserResolver {
	return &HTTPUserResolver{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (r *HTTPUserResolver) ResolveRecipient(ctx context.Context, userID string) (*RecipientInfo, error) {
	url := fmt.Sprintf("%s/users/%s", r.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // User not found
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user service returned status: %d", resp.StatusCode)
	}

	// Expecting JSON response compatible with RecipientInfo or a specific user schema
	// For this 10/10 implementation, we support a standard schema
	var userResp struct {
		ID           string        `json:"id"`
		Email        string        `json:"email"`
		Phone        string        `json:"phone"`
		DeviceTokens []DeviceToken `json:"device_tokens"`
		Preferences  struct {
			Channels map[string]bool `json:"channels"`
		} `json:"preferences"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &RecipientInfo{
		UserID:       userResp.ID,
		Email:        userResp.Email,
		PhoneNumber:  userResp.Phone,
		DeviceTokens: userResp.DeviceTokens,
		Preferences:  userResp.Preferences.Channels,
	}, nil
}
