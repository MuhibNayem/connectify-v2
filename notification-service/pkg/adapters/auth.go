package adapters

import "context"

// AuthenticationAdapter defines the interface for authenticating API requests.
// Implement this interface to integrate with your existing authentication system.
//
// Example implementations:
//   - JWT: Validate JWT tokens
//   - OAuth2: Validate OAuth2 bearer tokens
//   - API Key: Validate API keys from headers
//   - Session: Validate session cookies
//   - mTLS: Validate client certificates
//   - No-op: Allow all requests (for testing)
type AuthenticationAdapter interface {
	// Authenticate validates credentials and returns the authenticated user context.
	// Returns error if authentication fails.
	Authenticate(ctx context.Context, credentials interface{}) (*AuthContext, error)

	// ValidateToken validates an authentication token (JWT, API key, etc.)
	// Returns the authenticated user context if valid.
	ValidateToken(ctx context.Context, token string) (*AuthContext, error)
}

// AuthContext contains information about the authenticated user/client
type AuthContext struct {
	// UserID is the unique identifier for the authenticated user
	UserID string `json:"user_id"`

	// ClientID for service-to-service authentication
	ClientID string `json:"client_id,omitempty"`

	// Scopes/permissions granted to this user/client
	Scopes []string `json:"scopes,omitempty"`

	// TenantID for multi-tenant systems
	TenantID string `json:"tenant_id,omitempty"`

	// Custom metadata - use this for industry-specific fields
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HasScope checks if the auth context has a specific scope/permission
func (a *AuthContext) HasScope(scope string) bool {
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
