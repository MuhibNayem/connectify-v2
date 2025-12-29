package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GraphClient defines the interface for graph database operations
// This abstraction allows switching between Neo4j and Dgraph
type GraphClient interface {
	// SyncUser ensures a user exists in the graph
	SyncUser(ctx context.Context, userID primitive.ObjectID) error

	// SendRequest creates a friend request from one user to another
	SendRequest(ctx context.Context, from, to primitive.ObjectID) error

	// AcceptRequest accepts a friend request and creates friendship
	AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error

	// RejectRequest removes a friend request
	RejectRequest(ctx context.Context, from, to primitive.ObjectID) error

	// Unfriend removes a friendship between two users
	Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error

	// BlockUser blocks a user and removes any friendship/requests
	BlockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error

	// UnblockUser removes a block relationship
	UnblockUser(ctx context.Context, blocker, blocked primitive.ObjectID) error

	// CheckFriendshipStatus returns the full friendship status between two users
	CheckFriendshipStatus(ctx context.Context, me, other primitive.ObjectID) (areFriends, requestSent, requestReceived, blockedByMe, blockedByOther bool, err error)

	// AreFriends checks if two users are friends
	AreFriends(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error)

	// Block is an alias for BlockUser (for outbox processor compatibility)
	Block(ctx context.Context, blocker, blocked primitive.ObjectID) error

	// Unblock is an alias for UnblockUser (for outbox processor compatibility)
	Unblock(ctx context.Context, blocker, blocked primitive.ObjectID) error
}

// Ensure implementations satisfy the interface
var _ GraphClient = (*GraphRepository)(nil)
var _ GraphClient = (*DgraphRepository)(nil)
