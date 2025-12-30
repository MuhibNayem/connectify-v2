package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
)

func TestAuthenticator_GenerateAndValidateToken(t *testing.T) {
	authenticator := auth.NewAuthenticator(auth.Config{
		JWTSecret: "test-secret-key-12345",
		JWTIssuer: "notification-service",
	})

	// Generate token
	token, err := authenticator.GenerateToken("user-123", "tenant-456", "admin", time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	claims, err := authenticator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("Expected UserID 'user-123', got '%s'", claims.UserID)
	}
	if claims.TenantID != "tenant-456" {
		t.Errorf("Expected TenantID 'tenant-456', got '%s'", claims.TenantID)
	}
	if claims.Role != "admin" {
		t.Errorf("Expected Role 'admin', got '%s'", claims.Role)
	}
}

func TestAuthenticator_InvalidToken(t *testing.T) {
	authenticator := auth.NewAuthenticator(auth.Config{
		JWTSecret: "test-secret-key-12345",
		JWTIssuer: "notification-service",
	})

	_, err := authenticator.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestAuthenticator_APIKeyAuth(t *testing.T) {
	authenticator := auth.NewAuthenticator(auth.Config{
		JWTSecret: "test-secret",
		APIKeys: map[string]string{
			"valid-api-key-123": "events-service",
		},
	})

	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("Expected claims in context")
			return
		}
		if claims.UserID != "service:events-service" {
			t.Errorf("Expected service user ID, got '%s'", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/notifications", nil)
	req.Header.Set("X-API-Key", "valid-api-key-123")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestAuthenticator_InvalidAPIKey(t *testing.T) {
	authenticator := auth.NewAuthenticator(auth.Config{
		JWTSecret: "test-secret",
		APIKeys: map[string]string{
			"valid-api-key": "service",
		},
	})

	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/notifications", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestAuthenticator_SkipPaths(t *testing.T) {
	authenticator := auth.NewAuthenticator(auth.Config{
		JWTSecret: "test-secret",
	})

	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Health check should be skipped
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", rr.Code)
	}
}
