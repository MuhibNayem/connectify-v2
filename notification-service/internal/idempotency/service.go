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
// Uses Redis SETNX for atomic check-and-set
func (s *Service) Check(ctx context.Context, eventID string) (bool, error) {
	if s.client == nil {
		return false, nil // No Redis = no idempotency (dev mode)
	}

	key := "idempotency:" + eventID

	// SETNX: Set if Not eXists
	// Returns true if key was set (first time processing)
	// Returns false if key already exists (duplicate)
	wasSet, err := s.client.SetNX(ctx, key, "processed", s.ttl).Result()
	if err != nil {
		return false, err
	}

	// If wasSet is false, it means the key already existed = duplicate
	isDuplicate := !wasSet
	return isDuplicate, nil
}

// MarkProcessed explicitly marks an event as processed
// Useful for delayed marking after successful processing
func (s *Service) MarkProcessed(ctx context.Context, eventID string) error {
	if s.client == nil {
		return nil
	}

	key := "idempotency:" + eventID
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
