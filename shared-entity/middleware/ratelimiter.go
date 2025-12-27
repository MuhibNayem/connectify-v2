package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitObserver is invoked whenever a rate limit is triggered.
type RateLimitObserver func(action string)

func notifyRateLimitObserver(observer RateLimitObserver, action string) {
	if observer != nil {
		observer(action)
	}
}

// IPRateLimiter stores a rate limiter for each IP address
type IPRateLimiter struct {
	ips   map[string]*rate.Limiter
	mu    *sync.RWMutex
	limit rate.Limit
	burst int
}

// NewIPRateLimiter creates a new IPRateLimiter
func NewIPRateLimiter(limit rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		ips:   make(map[string]*rate.Limiter),
		mu:    &sync.RWMutex{},
		limit: limit,
		burst: burst,
	}
}

// AddIP creates a new rate limiter for an IP address
func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.limit, i.burst)
	i.ips[ip] = limiter
	return limiter
}

// GetLimiter returns the rate limiter for an IP address
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		return i.AddIP(ip)
	}

	return limiter
}

// RateLimiter is a middleware that limits the number of requests per IP
func RateLimiter(enabled bool, limit float64, burst int, action string, observer RateLimitObserver) gin.HandlerFunc {
	if !enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := NewIPRateLimiter(rate.Limit(limit), burst)

	return func(c *gin.Context) {
		ipLimiter := limiter.GetLimiter(c.ClientIP())
		if !ipLimiter.Allow() {
			notifyRateLimitObserver(observer, action)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}

		c.Next()
	}
}

// StrictRateLimiter creates a custom rate limiter middleware with specified limit and burst
func StrictRateLimiter(r float64, b int, action string, observer RateLimitObserver) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate.Limit(r), b)

	return func(c *gin.Context) {
		ipLimiter := limiter.GetLimiter(c.ClientIP())
		if !ipLimiter.Allow() {
			notifyRateLimitObserver(observer, action)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Please slow down."})
			return
		}

		c.Next()
	}
}

// EventRateLimitConfig defines rate limiting configuration for specific event actions
type EventRateLimitConfig struct {
	MaxRequests int           // Maximum requests allowed
	Window      time.Duration // Time window for rate limiting
	KeyPrefix   string        // Redis key prefix
	Action      string        // Action label for observability
}

// Predefined event rate limit configurations
var (
	// RSVPRateLimit - 10 requests per minute
	RSVPRateLimit = EventRateLimitConfig{
		MaxRequests: 10,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:rsvp",
		Action:      "events:rsvp",
	}

	// InviteRateLimit - 20 invitations per hour
	InviteRateLimit = EventRateLimitConfig{
		MaxRequests: 20,
		Window:      time.Hour,
		KeyPrefix:   "ratelimit:invite",
		Action:      "events:invite",
	}

	// EventPostRateLimit - 5 posts per minute
	EventPostRateLimit = EventRateLimitConfig{
		MaxRequests: 5,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:event_post",
		Action:      "events:posts",
	}

	// RecommendationRateLimit - 30 requests per minute
	RecommendationRateLimit = EventRateLimitConfig{
		MaxRequests: 30,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:recommendations",
		Action:      "events:recommendations",
	}

	// TrendingRateLimit - 30 requests per minute
	TrendingRateLimit = EventRateLimitConfig{
		MaxRequests: 30,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:trending",
		Action:      "events:trending",
	}

	// SearchRateLimit - 30 requests per minute
	SearchRateLimit = EventRateLimitConfig{
		MaxRequests: 30,
		Window:      time.Minute,
		KeyPrefix:   "ratelimit:search",
		Action:      "events:search",
	}

	// CreateEventRateLimit - 5 events per hour
	CreateEventRateLimit = EventRateLimitConfig{
		MaxRequests: 5,
		Window:      time.Hour,
		KeyPrefix:   "ratelimit:create_event",
		Action:      "events:create",
	}
)

// EventRateLimiter creates a Redis-based rate limiting middleware for event actions
func EventRateLimiter(redisClient *redis.ClusterClient, config EventRateLimitConfig, observer RateLimitObserver) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			// Fall back to IP-based rate limiting for unauthenticated requests
			userID = c.ClientIP()
		}

		key := fmt.Sprintf("%s:%v", config.KeyPrefix, userID)

		// Check current count
		countStr, err := redisClient.Get(c.Request.Context(), key)
		if err != nil && countStr == "" {
			// Key doesn't exist, start new window
			if err := redisClient.Set(c.Request.Context(), key, "1", config.Window); err != nil {
				// Redis error - allow request but log
				c.Next()
				return
			}
			c.Next()
			return
		}

		count, _ := strconv.Atoi(countStr)
		if count >= config.MaxRequests {
			// Rate limited
			retryAfter := int64(config.Window.Seconds())
			notifyRateLimitObserver(observer, config.Action)
			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
			c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     fmt.Sprintf("Too many requests. Please try again in %d seconds.", retryAfter),
				"retry_after": retryAfter,
			})
			return
		}

		// Increment counter
		newCount := count + 1
		if err := redisClient.Set(c.Request.Context(), key, strconv.Itoa(newCount), config.Window); err != nil {
			// Redis error - allow request
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(config.MaxRequests-newCount))

		c.Next()
	}
}
