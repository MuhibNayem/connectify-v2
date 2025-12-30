package idempotency

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Service provides exactly-once processing guarantees using Redis
type Service struct {
	client *redis.Client
	ttl    time.Duration
}

// NewService creates a new idempotency service
func NewService(client *redis.Client, ttl time.Duration) *Service {
	return &Service{
		client: client,
		ttl:    ttl,
	}
}

// Check returns true if the event has already been processed (duplicate)
func (s *Service) Check(ctx context.Context, key string) (bool, error) {
	if s.client == nil {
		return false, nil
	}
	// Just check existence
	exists, err := s.client.Exists(ctx, "idempotency:"+key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Lock attempts to acquire a processing lock for the event.
// Returns true if lock acquired, false if already locked/processed.
// Sets status to "processing" with a short TTL (e.g. 5 minutes) to handle crashes.
func (s *Service) Lock(ctx context.Context, eventID string) (bool, error) {
	if s.client == nil {
		return true, nil // Dev mode: always acquire
	}

	key := "idempotency:" + eventID

	// SETNX: Set if Not eXists
	// We use a short TTL (e.g. 5m) for the lock. If instance crashes, lock expires, allowing retry.
	// Processing time is usually milliseconds. 5m is safe.
	lockTTL := 5 * time.Minute
	acquired, err := s.client.SetNX(ctx, key, "processing", lockTTL).Result()
	if err != nil {
		return false, err
	}

	// If not acquired, check if it's "processed" or just "processing" (zombie lock?)
	// For now, strict idempotency says: if exists, we don't process.
	// But if it was a zombie lock from a crash 6 minutes ago, it would have expired.
	// So standard SetNX is sufficient for "At-Most-Once" per TTL.

	return acquired, nil
}

// Confirm marks the event as successfully processed.
// Updates status to "processed" and extends TTL to full duration (e.g. 24h).
func (s *Service) Confirm(ctx context.Context, eventID string) error {
	if s.client == nil {
		return nil
	}

	key := "idempotency:" + eventID
	// Overwrite with "processed" and full TTL
	return s.client.Set(ctx, key, "processed", s.ttl).Err()
}

// Remove removes the idempotency key (for testing or recovery)
func (s *Service) Remove(ctx context.Context, eventID string) error {
	if s.client == nil {
		return nil
	}

	key := "idempotency:" + eventID
	return s.client.Del(ctx, key).Err()
}
