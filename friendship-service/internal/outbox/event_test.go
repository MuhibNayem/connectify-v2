package outbox_test

import (
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/outbox"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewEvent_CreatesValidEvent(t *testing.T) {
	t.Parallel()

	aggregateID := primitive.NewObjectID()
	eventType := outbox.EventFriendRequestSent
	payload := bson.M{
		"requester_id": primitive.NewObjectID().Hex(),
		"receiver_id":  primitive.NewObjectID().Hex(),
	}

	event := outbox.NewEvent(aggregateID, eventType, payload)

	if event.ID.IsZero() {
		t.Error("Expected non-zero ID")
	}
	if event.AggregateID != aggregateID {
		t.Errorf("Expected AggregateID %s, got %s", aggregateID.Hex(), event.AggregateID.Hex())
	}
	if event.EventType != eventType {
		t.Errorf("Expected EventType %s, got %s", eventType, event.EventType)
	}
	if event.Status != outbox.StatusPending {
		t.Errorf("Expected Status %s, got %s", outbox.StatusPending, event.Status)
	}
	if event.RetryCount != 0 {
		t.Errorf("Expected RetryCount 0, got %d", event.RetryCount)
	}
	if event.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", event.MaxRetries)
	}
	if event.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt")
	}
	if event.ProcessedAt != nil {
		t.Error("Expected nil ProcessedAt")
	}
}

func TestNewEvent_NextRetry_IsNow(t *testing.T) {
	t.Parallel()

	before := time.Now().Add(-1 * time.Second)
	event := outbox.NewEvent(primitive.NewObjectID(), outbox.EventFriendRequestSent, bson.M{})
	after := time.Now().Add(1 * time.Second)

	if event.NextRetry.Before(before) || event.NextRetry.After(after) {
		t.Errorf("NextRetry should be approximately now, got: %v", event.NextRetry)
	}
}

func TestEventStatus_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   outbox.EventStatus
		expected string
	}{
		{"Pending", outbox.StatusPending, "pending"},
		{"Processed", outbox.StatusProcessed, "processed"},
		{"Failed", outbox.StatusFailed, "failed"},
		{"Dead", outbox.StatusDead, "dead"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestEventType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
	}{
		{outbox.EventFriendRequestSent, "FriendRequestSent"},
		{outbox.EventFriendRequestAccepted, "FriendRequestAccepted"},
		{outbox.EventFriendRequestRejected, "FriendRequestRejected"},
		{outbox.EventUnfriended, "Unfriended"},
		{outbox.EventUserBlocked, "UserBlocked"},
		{outbox.EventUserUnblocked, "UserUnblocked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.name)
			}
		})
	}
}

func TestEvent_Payload_ContainsRequiredFields(t *testing.T) {
	t.Parallel()

	requesterID := primitive.NewObjectID()
	receiverID := primitive.NewObjectID()

	payload := bson.M{
		"requester_id": requesterID.Hex(),
		"receiver_id":  receiverID.Hex(),
	}

	event := outbox.NewEvent(primitive.NewObjectID(), outbox.EventFriendRequestSent, payload)

	if event.Payload["requester_id"] != requesterID.Hex() {
		t.Errorf("Expected requester_id %s, got %s", requesterID.Hex(), event.Payload["requester_id"])
	}
	if event.Payload["receiver_id"] != receiverID.Hex() {
		t.Errorf("Expected receiver_id %s, got %s", receiverID.Hex(), event.Payload["receiver_id"])
	}
}
