package mocks

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GraphRepository is a mock implementation of the graph repository
type GraphRepository struct {
	SendRequestFunc           func(ctx context.Context, from, to primitive.ObjectID) error
	AcceptRequestFunc         func(ctx context.Context, from, to primitive.ObjectID) error
	RejectRequestFunc         func(ctx context.Context, from, to primitive.ObjectID) error
	UnfriendFunc              func(ctx context.Context, user1, user2 primitive.ObjectID) error
	BlockFunc                 func(ctx context.Context, blocker, blocked primitive.ObjectID) error
	UnblockFunc               func(ctx context.Context, blocker, blocked primitive.ObjectID) error
	AreFriendsFunc            func(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error)
	CheckFriendshipStatusFunc func(ctx context.Context, me, other primitive.ObjectID) (bool, bool, bool, bool, bool, error)
}

func (m *GraphRepository) SendRequest(ctx context.Context, from, to primitive.ObjectID) error {
	if m.SendRequestFunc != nil {
		return m.SendRequestFunc(ctx, from, to)
	}
	return nil
}

func (m *GraphRepository) AcceptRequest(ctx context.Context, from, to primitive.ObjectID) error {
	if m.AcceptRequestFunc != nil {
		return m.AcceptRequestFunc(ctx, from, to)
	}
	return nil
}

func (m *GraphRepository) RejectRequest(ctx context.Context, from, to primitive.ObjectID) error {
	if m.RejectRequestFunc != nil {
		return m.RejectRequestFunc(ctx, from, to)
	}
	return nil
}

func (m *GraphRepository) Unfriend(ctx context.Context, user1, user2 primitive.ObjectID) error {
	if m.UnfriendFunc != nil {
		return m.UnfriendFunc(ctx, user1, user2)
	}
	return nil
}

func (m *GraphRepository) Block(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	if m.BlockFunc != nil {
		return m.BlockFunc(ctx, blocker, blocked)
	}
	return nil
}

func (m *GraphRepository) Unblock(ctx context.Context, blocker, blocked primitive.ObjectID) error {
	if m.UnblockFunc != nil {
		return m.UnblockFunc(ctx, blocker, blocked)
	}
	return nil
}

func (m *GraphRepository) AreFriends(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error) {
	if m.AreFriendsFunc != nil {
		return m.AreFriendsFunc(ctx, user1, user2)
	}
	return false, nil
}

func (m *GraphRepository) CheckFriendshipStatus(ctx context.Context, me, other primitive.ObjectID) (bool, bool, bool, bool, bool, error) {
	if m.CheckFriendshipStatusFunc != nil {
		return m.CheckFriendshipStatusFunc(ctx, me, other)
	}
	return false, false, false, false, false, nil
}
