package ratelimit

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/auth"
	"github.com/redis/go-redis/v9"
)

// Config for rate limiting
type Config struct {
	RequestsPerSecond int
	BurstSize         int
	KeyPrefix         string
}

// Limiter implements Redis-based rate limiting
type Limiter struct {
	client *redis.Client
	config Config
}

// NewLimiter creates a new rate limiter
func NewLimiter(client *redis.Client, config Config) *Limiter {
	if config.RequestsPerSecond == 0 {
		config.RequestsPerSecond = 100 // default
	}
	if config.BurstSize == 0 {
		config.BurstSize = 200 // default
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}

	return &Limiter{
		client: client,
		config: config,
	}
}

// Allow checks if a request is allowed under rate limits
// Uses the sliding window counter algorithm
func (l *Limiter) Allow(ctx context.Context, key string) (bool, int, int) {
	if l.client == nil {
		return true, l.config.RequestsPerSecond, l.config.RequestsPerSecond
	}

	fullKey := l.config.KeyPrefix + ":" + key
	now := time.Now().Unix()
	windowKey := fullKey + ":" + strconv.FormatInt(now, 10)

	// Increment counter for current second
	count, err := l.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return true, l.config.RequestsPerSecond, l.config.RequestsPerSecond // Fail open
	}

	// Set TTL on first request in this second
	if count == 1 {
		l.client.Expire(ctx, windowKey, 2*time.Second)
	}

	remaining := l.config.RequestsPerSecond - int(count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := count <= int64(l.config.RequestsPerSecond)
	return allowed, l.config.RequestsPerSecond, remaining
}

// Middleware returns HTTP middleware for rate limiting
func (l *Limiter) Middleware(keyExtractor func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyExtractor(r)
			allowed, limit, remaining := l.Allow(r.Context(), key)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyExtractor extracts client IP for rate limiting
func IPKeyExtractor(r *http.Request) string {
	// 1. Check X-Forwarded-For (standard proxy header)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF can be comma separated list: "client, proxy1, proxy2"
		// We want the client IP (first one)
		if comma := strings.Index(xff, ","); comma != -1 {
			return strings.TrimSpace(xff[:comma])
		}
		return xff
	}

	// 2. Check X-Real-IP (common alternative)
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// 3. Fallback to RemoteAddr, stripping port
	addr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// UserKeyExtractor extracts user ID for rate limiting
func UserKeyExtractor(r *http.Request) string {
	// Use the auth package helper to extract UserID
	if userID := auth.UserIDFromContext(r.Context()); userID != "" {
		return userID
	}
	// Fallback to IP if not authenticated
	return IPKeyExtractor(r)
}
