package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPUserResolver implements the UserResolver interface by making HTTP requests to an external user service.
// It is responsible for initializing the HTTP client and handling the base URL configuration.
type HTTPUserResolver struct {
	client  *http.Client
	baseURL string
}

// NewHTTPUserResolver creates a new instance of HTTPUserResolver with the specified base URL.
// It sets up an HTTP client with a default timeout of 5 seconds to ensure resilience.
func NewHTTPUserResolver(baseURL string) *HTTPUserResolver {
	return &HTTPUserResolver{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: baseURL,
	}
}

// ResolveRecipient fetches user details given a unique User ID.
// It performs a GET request to the user service and maps the response to the RecipientInfo struct.
// Returns a detailed error if the request fails or the status code is not 200 OK.
// Returns nil, nil if the user is not found (404).
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

// GetPreferences retrieves the communication preferences for a specific user.
// It piggybacks on the user details endpoint to minimize network calls if preferences are embedded.
// Returns default or nil preferences if not explicitly set or found.
func (r *HTTPUserResolver) GetPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	mainUrl := fmt.Sprintf("%s/users/%s", r.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, "GET", mainUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Return nil to use defaults
	}

	var userResp struct {
		Preferences UserPreferences `json:"preferences"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode preferences: %w", err)
	}

	return &userResp.Preferences, nil
}

// UpdatePreferences updates the user's communication preferences.
// This implementation currently returns nil as the core delivery service usually only reads preferences.
func (r *HTTPUserResolver) UpdatePreferences(ctx context.Context, userID string, prefs *UserPreferences) error {
	return nil
}
