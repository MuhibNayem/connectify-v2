package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Authenticator handles both API Key and JWT authentication
// This enables the service to work as:
// 1. Internal microservice (API Key from other services)
// 2. External API (JWT from end users/frontends)
// Authenticator handles both API Key and JWT authentication
// Supports both HS256 (Shared Secret) and RS256 (Public Key)
type Authenticator struct {
	secretKey    []byte
	publicKey    *rsa.PublicKey
	issuer       string
	skipPaths    map[string]bool
	validAPIKeys map[string]string // key -> service name
}

// Config for the authenticator
type Config struct {
	JWTSecret    string
	JWTPublicKey string // PEM content or file path
	JWTIssuer    string
	APIKeys      map[string]string // Maps API keys to service names
}

// NewAuthenticator creates a new authenticator with dual-mode support
func NewAuthenticator(cfg Config) *Authenticator {
	auth := &Authenticator{
		secretKey: []byte(cfg.JWTSecret),
		issuer:    cfg.JWTIssuer,
		skipPaths: map[string]bool{
			"/health":  true,
			"/metrics": true,
		},
		validAPIKeys: cfg.APIKeys,
	}

	// Try to parse Public Key if provided
	if cfg.JWTPublicKey != "" {
		// content or file? Check if file exists
		var content []byte
		if _, err := os.Stat(cfg.JWTPublicKey); err == nil {
			content, _ = os.ReadFile(cfg.JWTPublicKey)
		} else {
			content = []byte(cfg.JWTPublicKey)
		}

		block, _ := pem.Decode(content)
		if block != nil {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err == nil {
				if rsaPub, ok := pub.(*rsa.PublicKey); ok {
					auth.publicKey = rsaPub
				}
			}
		}
	}

	return auth
}

// Middleware returns HTTP middleware for dual-mode authentication
// Priority: 1. API Key (X-API-Key header) for service-to-service
//  2. JWT (Authorization: Bearer) for client-to-service
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for whitelisted paths
		if a.skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// MODE 1: Check API Key first (Service-to-Service)
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if serviceName, valid := a.validAPIKeys[apiKey]; valid {
				// Create service-level claims
				claims := &Claims{
					UserID:   "service:" + serviceName,
					TenantID: "system",
					Role:     "service",
				}
				ctx := ContextWithClaims(r.Context(), claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
			return
		}

		// MODE 2: JWT Token (Client-to-Service)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		claims, err := a.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, `{"error":"invalid token: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Add claims to context
		ctx := ContextWithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateToken validates a JWT token and returns claims
func (a *Authenticator) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 1. Check if HMAC (HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			if len(a.secretKey) == 0 {
				return nil, fmt.Errorf("HS256 token received but no secret key configured")
			}
			return a.secretKey, nil
		}

		// 2. Check if RSA (RS256)
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			if a.publicKey == nil {
				return nil, fmt.Errorf("RS256 token received but no public key configured")
			}
			return a.publicKey, nil
		}

		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Validate issuer
		if claims.Issuer != a.issuer && a.issuer != "" {
			return nil, errors.New("invalid issuer")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateToken generates a JWT token (for testing/internal use)
func (a *Authenticator) GenerateToken(userID, tenantID, role string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secretKey)
}

// Context key type
type claimsContextKey struct{}

// ContextWithClaims adds claims to context
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext retrieves claims from context
func ClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(claimsContextKey{}).(*Claims); ok {
		return claims
	}
	return nil
}

// UserIDFromContext gets user ID from context
func UserIDFromContext(ctx context.Context) string {
	if claims := ClaimsFromContext(ctx); claims != nil {
		return claims.UserID
	}
	return ""
}

// TenantIDFromContext gets tenant ID from context
func TenantIDFromContext(ctx context.Context) string {
	if claims := ClaimsFromContext(ctx); claims != nil {
		return claims.TenantID
	}
	return ""
}
