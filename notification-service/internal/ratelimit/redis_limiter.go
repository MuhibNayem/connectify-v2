package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements token bucket algorithm using Redis
type RedisRateLimiter struct {
	client         *redis.Client
	maxTokens      int64
	refillRate     int64
	windowDuration time.Duration
}

func NewRedisRateLimiter(client *redis.Client, maxPerHour int64) *RedisRateLimiter {
	windowDuration := time.Hour
	return &RedisRateLimiter{
		client:         client,
		maxTokens:      maxPerHour,
		refillRate:     maxPerHour / int64(windowDuration.Seconds()),
		windowDuration: windowDuration,
	}
}

// Allow checks if a request is allowed for the given user
func (r *RedisRateLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:user:%s", userID)
	now := time.Now().Unix()

	// Lua script for atomic token bucket operations
	script := `
		local key = KEYS[1]
		local max_tokens = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or max_tokens
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add
		local elapsed = now - last_refill
		local refill = math.floor(elapsed * refill_rate)
		tokens = math.min(max_tokens, tokens + refill)
		
		-- Check if request allowed
		if tokens >= 1 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 3600)
			return 1
		else
			return 0
		end
	`

	result, err := r.client.Eval(ctx, script, []string{key},
		r.maxTokens, r.refillRate, now).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}

// GetRemaining returns remaining tokens for a user
func (r *RedisRateLimiter) GetRemaining(ctx context.Context, userID string) (int64, error) {
	key := fmt.Sprintf("ratelimit:user:%s", userID)

	tokens, err := r.client.HGet(ctx, key, "tokens").Int64()
	if err == redis.Nil {
		return r.maxTokens, nil
	}
	if err != nil {
		return 0, err
	}

	if tokens < 0 {
		return 0, nil
	}
	return tokens, nil
}

// Reset resets the rate limit for a user
func (r *RedisRateLimiter) Reset(ctx context.Context, userID string) error {
	key := fmt.Sprintf("ratelimit:user:%s", userID)
	return r.client.Del(ctx, key).Err()
}
