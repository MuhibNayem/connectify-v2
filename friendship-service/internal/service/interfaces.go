package service

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FriendshipRepository defines the interface for friendship persistence
type FriendshipRepository interface {
	CreateFriendship(ctx context.Context, friendship *models.Friendship) error
	GetFriendshipByID(ctx context.Context, id primitive.ObjectID) (*models.Friendship, error)
	GetFriendshipByUsers(ctx context.Context, userID1, userID2 primitive.ObjectID) (*models.Friendship, error)
	UpdateFriendshipStatus(ctx context.Context, id primitive.ObjectID, status models.FriendshipStatus) error
	ListFriendships(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error)
	Unfriend(ctx context.Context, userID, friendID primitive.ObjectID) error
	GetMutualFriends(ctx context.Context, userID1, userID2 primitive.ObjectID) ([]primitive.ObjectID, error)
	AddToBlockList(ctx context.Context, blockerID, blockedID primitive.ObjectID) error
	RemoveFromBlockList(ctx context.Context, blockerID, blockedID primitive.ObjectID) error
	GetBlockedUsers(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error)
	IsBlocked(ctx context.Context, userID, otherUserID primitive.ObjectID) (bool, bool, error)
	SearchFriends(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error)
}

// GraphRepository defines the interface for graph operations (Neo4j)
type GraphRepository interface {
	CreateFriendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) error
	AcceptFriendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) error
	RejectFriendRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) error
	RemoveFriendship(ctx context.Context, userID, friendID primitive.ObjectID) error
	BlockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error
	UnblockUser(ctx context.Context, blockerID, blockedID primitive.ObjectID) error
}

// UserClient defines the interface for user service interactions
type UserClient interface {
	AddFriend(ctx context.Context, userID, friendID primitive.ObjectID) error
	RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error
}

// EventProducer defines the interface for publishing friendship events
type EventProducer interface {
	Publish(ctx context.Context, eventType string, userID primitive.ObjectID, payload interface{}) error
	Close() error
}

// FriendshipCache defines the interface for caching friendship data
type FriendshipCache interface {
	GetFriendshipStatus(ctx context.Context, userID1, userID2 primitive.ObjectID) (*models.FriendshipStatus, error)
	SetFriendshipStatus(ctx context.Context, userID1, userID2 primitive.ObjectID, status models.FriendshipStatus) error
	InvalidateFriendship(ctx context.Context, userID1, userID2 primitive.ObjectID) error
	GetBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID) (*bool, error)
	SetBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID, blocked bool) error
	InvalidateBlockStatus(ctx context.Context, blockerID, blockedID primitive.ObjectID) error
}
