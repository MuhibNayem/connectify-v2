package events

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserUpdatedEvent represents a change in user profile data.
type UserUpdatedEvent struct {
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	FullName    string     `json:"full_name"`
	Avatar      string     `json:"avatar"`
	DateOfBirth *time.Time `json:"date_of_birth"`
}

// FriendshipEvent represents a change in friendship status.
type FriendshipEvent struct {
	RequesterID string    `json:"requester_id"`
	ReceiverID  string    `json:"receiver_id"`
	Status      string    `json:"status"`
	Action      string    `json:"action"` // "request", "accept", "reject", "remove", "block", "unblock"
	Timestamp   time.Time `json:"timestamp"`
}

// NotificationCreatedEvent represents a new notification to be persisted and delivered.
type NotificationCreatedEvent struct {
	ID          primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	RecipientID primitive.ObjectID     `json:"recipient_id" bson:"recipient_id"`
	SenderID    primitive.ObjectID     `json:"sender_id" bson:"sender_id"`
	Type        string                 `json:"type" bson:"type"` // Converted from enum to string for transport
	TargetID    primitive.ObjectID     `json:"target_id" bson:"target_id"`
	TargetType  string                 `json:"target_type" bson:"target_type"`
	Content     string                 `json:"content" bson:"content"`
	Data        map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`
	Read        bool                   `json:"read" bson:"read"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
}
