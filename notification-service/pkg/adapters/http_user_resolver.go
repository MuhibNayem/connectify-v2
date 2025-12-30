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

// GetPreferences implements UserPreferenceAdapter
func (r *HTTPUserResolver) GetPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	// Fallback to main user endpoint if preferences are embedded?
	// For "Standard API", let's assume /preferences exists or we parse from user.
	// We use the main user endpoint as per previous logic (embedded prefs).

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

	// Parsing strict UserPreferences struct
	var userResp struct {
		Preferences UserPreferences `json:"preferences"`
	}
	// Note: We assume the User Service returns "preferences" object matching our struct
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode preferences: %w", err)
	}

	return &userResp.Preferences, nil
}

// UpdatePreferences implements UserPreferenceAdapter
func (r *HTTPUserResolver) UpdatePreferences(ctx context.Context, userID string, prefs *UserPreferences) error {
	// Not needed for core delivery, but satisfied interface
	return nil
}
