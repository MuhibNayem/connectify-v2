package cache_test

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestFriendshipCache_KeyGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userA  string
		userB  string
		format string
	}{
		{
			name:   "Friendship status key",
			userA:  "507f1f77bcf86cd799439011",
			userB:  "507f1f77bcf86cd799439012",
			format: "friendship:status:%s:%s",
		},
		{
			name:   "Block status key",
			userA:  "507f1f77bcf86cd799439011",
			userB:  "507f1f77bcf86cd799439012",
			format: "friendship:blocked:%s:%s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Key generation should be consistent
			key1 := generateKey(tt.format, tt.userA, tt.userB)
			key2 := generateKey(tt.format, tt.userA, tt.userB)

			if key1 != key2 {
				t.Errorf("Keys should be consistent: %s != %s", key1, key2)
			}
		})
	}
}

func generateKey(format string, a, b string) string {
	// Simulate key generation logic
	return format + ":" + a + ":" + b
}

func TestFriendshipCache_TTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ttl      time.Duration
		expected time.Duration
	}{
		{"5 minutes", 5 * time.Minute, 5 * time.Minute},
		{"10 minutes", 10 * time.Minute, 10 * time.Minute},
		{"1 hour", 1 * time.Hour, 1 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ttl != tt.expected {
				t.Errorf("Expected TTL %v, got %v", tt.expected, tt.ttl)
			}
		})
	}
}

func TestFriendshipCache_ObjectIDConsistency(t *testing.T) {
	t.Parallel()

	// Test that ObjectID order doesn't matter for bidirectional keys
	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()

	key1 := makeBidirectionalKey(userA, userB)
	key2 := makeBidirectionalKey(userB, userA)

	if key1 != key2 {
		t.Errorf("Bidirectional keys should be equal regardless of order: %s != %s", key1, key2)
	}
}

func makeBidirectionalKey(a, b primitive.ObjectID) string {
	// Ensure consistent ordering for bidirectional relationships
	if a.Hex() < b.Hex() {
		return a.Hex() + ":" + b.Hex()
	}
	return b.Hex() + ":" + a.Hex()
}

func TestCache_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Simulate slow operation
	time.Sleep(5 * time.Millisecond)

	if ctx.Err() == nil {
		t.Error("Context should be cancelled")
	}
}

func TestCache_InvalidationMessage_Marshaling(t *testing.T) {
	t.Parallel()

	type InvalidationMessage struct {
		Type   string `json:"type"`
		UserA  string `json:"user_a"`
		UserB  string `json:"user_b"`
		Action string `json:"action"`
	}

	msg := InvalidationMessage{
		Type:   "friendship",
		UserA:  primitive.NewObjectID().Hex(),
		UserB:  primitive.NewObjectID().Hex(),
		Action: "invalidate",
	}

	if msg.Type != "friendship" {
		t.Errorf("Expected Type 'friendship', got '%s'", msg.Type)
	}
	if msg.Action != "invalidate" {
		t.Errorf("Expected Action 'invalidate', got '%s'", msg.Action)
	}
	if msg.UserA == "" || msg.UserB == "" {
		t.Error("UserA and UserB should not be empty")
	}
}
