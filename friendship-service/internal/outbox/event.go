package outbox

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Event represents an outbox event for eventual consistency
type Event struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	AggregateID primitive.ObjectID `bson:"aggregate_id"`
	EventType   string             `bson:"event_type"`
	Payload     bson.M             `bson:"payload"`
	Status      EventStatus        `bson:"status"`
	RetryCount  int                `bson:"retry_count"`
	MaxRetries  int                `bson:"max_retries"`
	NextRetry   time.Time          `bson:"next_retry"`
	Error       string             `bson:"error,omitempty"`
	CreatedAt   time.Time          `bson:"created_at"`
	ProcessedAt *time.Time         `bson:"processed_at,omitempty"`
}

type EventStatus string

const (
	StatusPending   EventStatus = "pending"
	StatusProcessed EventStatus = "processed"
	StatusFailed    EventStatus = "failed"
	StatusDead      EventStatus = "dead" // Exceeded max retries
)

// Event types
const (
	EventFriendRequestSent     = "FriendRequestSent"
	EventFriendRequestAccepted = "FriendRequestAccepted"
	EventFriendRequestRejected = "FriendRequestRejected"
	EventUnfriended            = "Unfriended"
	EventUserBlocked           = "UserBlocked"
	EventUserUnblocked         = "UserUnblocked"
)

// NewEvent creates a new outbox event
func NewEvent(aggregateID primitive.ObjectID, eventType string, payload bson.M) *Event {
	return &Event{
		ID:          primitive.NewObjectID(),
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payload,
		Status:      StatusPending,
		RetryCount:  0,
		MaxRetries:  5,
		NextRetry:   time.Now(),
		CreatedAt:   time.Now(),
	}
}
