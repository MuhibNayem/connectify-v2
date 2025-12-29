package repository

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GraphClient defines the interface for graph database operations in events-service
type GraphClient interface {
	// AddAttendee adds a relationship (:User)-[:GOING]->(:Event)
	AddAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error

	// RemoveAttendee removes the relationship (:User)-[:GOING]->(:Event)
	RemoveAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error

	// GetFriendsGoing returns IDs of friends who are going to the event
	GetFriendsGoing(ctx context.Context, userID, eventID primitive.ObjectID) ([]string, error)

	// GetRecommendedEventsFromGraph returns FB-scale personalized event recommendations
	GetRecommendedEventsFromGraph(ctx context.Context, userID string, limit int) ([]service.GraphRecommendation, error)

	// AddUserInterest tracks user interest in event categories
	AddUserInterest(ctx context.Context, userID, category string) error

	// SetEventCategory links an event to a category for interest matching
	SetEventCategory(ctx context.Context, eventID, category string) error

	// AddFriendship creates bidirectional FRIEND relationship
	AddFriendship(ctx context.Context, userID1, userID2 string) error

	// RemoveFriendship removes FRIEND relationship
	RemoveFriendship(ctx context.Context, userID1, userID2 string) error

	// GetMutualFriendsCount returns count of mutual friends between user and event host
	GetMutualFriendsCount(ctx context.Context, userID, hostID string) (int, error)
}

// Ensure implementations satisfy the interface
var _ GraphClient = (*EventGraphRepository)(nil)
var _ GraphClient = (*DgraphRepository)(nil) // Uncomment when DgraphRepository is implemented
