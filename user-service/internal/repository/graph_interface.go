package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GraphClient defines the interface for graph database operations in user-service
type GraphClient interface {
	// SyncUser ensures a user exists in the graph
	SyncUser(ctx context.Context, userID primitive.ObjectID) error

	// SendRequest creates a friend request relationship
	SendRequest(ctx context.Context, from, to primitive.ObjectID) error

	// AcceptRequest transitions a request to a friendship
	AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error

	// RejectRequest removes a friend request
	RejectRequest(ctx context.Context, from, to primitive.ObjectID) error

	// Unfriend removes a friendship relationship
	Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error

	// BlockUser creates a block relationship and removes any existing friendship/request
	BlockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error

	// UnblockUser removes a block relationship
	UnblockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error

	// GetFriendIDs returns a list of friend user IDs
	GetFriendIDs(ctx context.Context, userID primitive.ObjectID) ([]string, error)
}

// Ensure implementations satisfy the interface
var _ GraphClient = (*GraphRepository)(nil)

var _ GraphClient = (*DgraphRepository)(nil) // Uncomment when DgraphRepository is implemented
