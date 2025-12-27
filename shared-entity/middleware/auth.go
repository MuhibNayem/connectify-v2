package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// ErrRevocationCheckFailed indicates we could not verify the token's revocation status.
var ErrRevocationCheckFailed = errors.New("revocation check failed")

// TokenBlacklist defines the interface for checking/adding token blacklist
// This allows any Redis client type (single, cluster, sentinel) to be used
type TokenBlacklist interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

type authSettings struct {
	failClosed     bool
	failStatusCode int
	failMessage    string
}

// AuthOption configures optional behavior for the auth middleware.
type AuthOption func(*authSettings)

// WithFailClosed enables fail-closed behavior when the blacklist cannot be checked.
func WithFailClosed() AuthOption {
	return func(s *authSettings) {
		s.failClosed = true
	}
}

// WithFailClosedResponse enables fail-closed behavior and customizes the HTTP response for that scenario.
func WithFailClosedResponse(statusCode int, message string) AuthOption {
	return func(s *authSettings) {
		s.failClosed = true
		s.failStatusCode = statusCode
		s.failMessage = message
	}
}

// AuthMiddleware creates a Gin middleware for JWT authentication
// blacklist can be nil for stateless JWT validation (no revocation support)
func AuthMiddleware(jwtSecret string, blacklist TokenBlacklist, opts ...AuthOption) gin.HandlerFunc {
	settings := authSettings{
		failStatusCode: http.StatusServiceUnavailable,
		failMessage:    "authentication temporarily unavailable",
	}
	for _, opt := range opts {
		opt(&settings)
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		userID, err := validateTokenWithBlacklist(authHeader, jwtSecret, blacklist, settings.failClosed)
		if err != nil {
			if settings.failClosed && errors.Is(err, ErrRevocationCheckFailed) {
				status := settings.failStatusCode
				if status == 0 {
					status = http.StatusServiceUnavailable
				}
				message := settings.failMessage
				if message == "" {
					message = "authentication unavailable"
				}
				c.AbortWithStatusJSON(status, gin.H{"error": message})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("userID", userID)
		c.Set("user_id", userID) // Also set with underscore for compatibility
		c.Next()
	}
}

// JWTAuthSimple creates a simple JWT auth middleware without blacklist checking
// Use this when you don't have Redis available or don't need token revocation
func JWTAuthSimple(jwtSecret string) gin.HandlerFunc {
	return AuthMiddleware(jwtSecret, nil)
}

const wsAuthProtocolName = "connectify.auth"

func WSJwtAuthMiddleware(jwtSecret string, blacklist TokenBlacklist, opts ...AuthOption) gin.HandlerFunc {
	settings := authSettings{}
	for _, opt := range opts {
		opt(&settings)
	}
	return func(c *gin.Context) {
		tokenString, err := extractWebsocketToken(c.GetHeader("Sec-WebSocket-Protocol"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		userID, err := validateTokenWithBlacklist(tokenString, jwtSecret, blacklist, settings.failClosed)
		if err != nil {
			fmt.Printf("WS Auth Error: %v\n", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("userID", userID)
		c.Set("user_id", userID)
		c.Next()
	}
}

func extractWebsocketToken(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("Sec-WebSocket-Protocol header required")
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || trimmed == wsAuthProtocolName {
			continue
		}
		return trimmed, nil
	}
	return "", fmt.Errorf("websocket auth token missing")
}

// ValidateTokenWithBlacklist validates a JWT token with optional blacklist check
// If blacklist is nil, only JWT signature validation is performed
func ValidateTokenWithBlacklist(tokenString, jwtSecret string, blacklist TokenBlacklist) (string, error) {
	return validateTokenWithBlacklist(tokenString, jwtSecret, blacklist, false)
}

func validateTokenWithBlacklist(tokenString, jwtSecret string, blacklist TokenBlacklist, failClosed bool) (string, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("bearer token required")
	}

	// Check token blacklist only if Redis is available
	if blacklist != nil {
		_, err := blacklist.Get(context.Background(), "blacklist:"+tokenString).Result()
		if err == nil {
			return "", fmt.Errorf("token revoked")
		} else if err != redis.Nil {
			if failClosed {
				return "", fmt.Errorf("%w: %v", ErrRevocationCheckFailed, err)
			}
		}
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["type"] != "access" {
			return "", fmt.Errorf("invalid token type")
		}

		userID, ok := claims["id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid token claims")
		}

		return userID, nil
	}

	return "", fmt.Errorf("invalid token")
}

// ValidateToken is a backwards-compatible wrapper for existing code
// Deprecated: Use ValidateTokenWithBlacklist instead
func ValidateToken(tokenString, jwtSecret string, redisClient *redis.ClusterClient) (string, error) {
	return ValidateTokenWithBlacklist(tokenString, jwtSecret, redisClient)
}

// BlacklistToken adds a token to the Redis blacklist
func BlacklistToken(tokenString string, expiration time.Duration, blacklist TokenBlacklist) error {
	if blacklist == nil {
		return fmt.Errorf("blacklist not available")
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return fmt.Errorf("empty token")
	}

	ctx := context.Background()
	return blacklist.Set(ctx, "blacklist:"+tokenString, "1", expiration).Err()
}
