package mocks

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FriendshipCache is a mock implementation of the friendship cache
type FriendshipCache struct {
	GetFriendshipStatusFunc  func(ctx context.Context, user1, user2 primitive.ObjectID) (string, bool)
	SetFriendshipStatusFunc  func(ctx context.Context, user1, user2 primitive.ObjectID, status string, ttl time.Duration) error
	GetBlockStatusFunc       func(ctx context.Context, blocker, blocked primitive.ObjectID) (bool, bool)
	SetBlockStatusFunc       func(ctx context.Context, blocker, blocked primitive.ObjectID, isBlocked bool, ttl time.Duration) error
	InvalidateFriendshipFunc func(ctx context.Context, user1, user2 primitive.ObjectID)
	InvalidateBlockFunc      func(ctx context.Context, blocker, blocked primitive.ObjectID)
}

func (m *FriendshipCache) GetFriendshipStatus(ctx context.Context, user1, user2 primitive.ObjectID) (string, bool) {
	if m.GetFriendshipStatusFunc != nil {
		return m.GetFriendshipStatusFunc(ctx, user1, user2)
	}
	return "", false
}

func (m *FriendshipCache) SetFriendshipStatus(ctx context.Context, user1, user2 primitive.ObjectID, status string, ttl time.Duration) error {
	if m.SetFriendshipStatusFunc != nil {
		return m.SetFriendshipStatusFunc(ctx, user1, user2, status, ttl)
	}
	return nil
}

func (m *FriendshipCache) GetBlockStatus(ctx context.Context, blocker, blocked primitive.ObjectID) (bool, bool) {
	if m.GetBlockStatusFunc != nil {
		return m.GetBlockStatusFunc(ctx, blocker, blocked)
	}
	return false, false
}

func (m *FriendshipCache) SetBlockStatus(ctx context.Context, blocker, blocked primitive.ObjectID, isBlocked bool, ttl time.Duration) error {
	if m.SetBlockStatusFunc != nil {
		return m.SetBlockStatusFunc(ctx, blocker, blocked, isBlocked, ttl)
	}
	return nil
}

func (m *FriendshipCache) InvalidateFriendship(ctx context.Context, user1, user2 primitive.ObjectID) {
	if m.InvalidateFriendshipFunc != nil {
		m.InvalidateFriendshipFunc(ctx, user1, user2)
	}
}

func (m *FriendshipCache) InvalidateBlock(ctx context.Context, blocker, blocked primitive.ObjectID) {
	if m.InvalidateBlockFunc != nil {
		m.InvalidateBlockFunc(ctx, blocker, blocked)
	}
}
