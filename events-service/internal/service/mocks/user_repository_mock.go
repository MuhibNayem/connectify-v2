package mocks

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/integration"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockUserRepo is a mock implementation of UserRepo
type MockUserRepo struct {
	FindByIDFunc            func(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error)
	FindByIDsFunc           func(ctx context.Context, ids []primitive.ObjectID) ([]integration.EventUser, error)
	FindFriendBirthdaysFunc func(ctx context.Context, friendIDs []primitive.ObjectID) ([]integration.EventUser, []integration.EventUser, error)
	GetFriendsFunc          func(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error)
}

func (m *MockUserRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, nil // Return nil if not mocked, avoid panic but might cause logic issues if unchecked
}

func (m *MockUserRepo) FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]integration.EventUser, error) {
	if m.FindByIDsFunc != nil {
		return m.FindByIDsFunc(ctx, ids)
	}
	return []integration.EventUser{}, nil
}

func (m *MockUserRepo) FindFriendBirthdays(ctx context.Context, friendIDs []primitive.ObjectID) ([]integration.EventUser, []integration.EventUser, error) {
	if m.FindFriendBirthdaysFunc != nil {
		return m.FindFriendBirthdaysFunc(ctx, friendIDs)
	}
	return nil, nil, nil
}

func (m *MockUserRepo) GetFriends(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	if m.GetFriendsFunc != nil {
		return m.GetFriendsFunc(ctx, userID)
	}
	return []primitive.ObjectID{}, nil
}
