package integration

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventUser represents the subset of User data needed by the Events Service.
// This follows the "Bounded Context" pattern where the Event domain definition of a User
// is distinct from the full Authentication/Profile definition.
type EventUser struct {
	ID                   primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username             string               `bson:"username" json:"username"`
	FullName             string               `bson:"full_name,omitempty" json:"full_name,omitempty"`
	Avatar               string               `bson:"avatar" json:"avatar"`
	DateOfBirth          *time.Time           `bson:"date_of_birth,omitempty" json:"date_of_birth,omitempty"`
	Friends              []primitive.ObjectID `bson:"friends,omitempty" json:"friends,omitempty"`
	NotificationSettings NotificationSettings `bson:"notification_settings,omitempty" json:"notification_settings,omitempty"`
}

type NotificationSettings struct {
	NotifyOnEventInvite bool `bson:"notify_on_event_invite" json:"notify_on_event_invite"`
}

type EventFriendship struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RequesterID primitive.ObjectID `bson:"requester_id" json:"requester_id"`
	ReceiverID  primitive.ObjectID `bson:"receiver_id" json:"receiver_id"`
	Status      string             `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
