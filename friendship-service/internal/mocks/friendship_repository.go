package mocks

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FriendshipRepository is a mock implementation of the friendship repository
type FriendshipRepository struct {
	CreateRequestFunc           func(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error)
	UpdateStatusFunc            func(ctx context.Context, friendshipID, receiverID primitive.ObjectID, status models.FriendshipStatus) error
	FindByUsersFunc             func(ctx context.Context, user1, user2 primitive.ObjectID) (*models.Friendship, error)
	AreFriendsFunc              func(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error)
	GetFriendshipsFunc          func(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error)
	DeleteFriendshipFunc        func(ctx context.Context, user1, user2 primitive.ObjectID) error
	GetByIDFunc                 func(ctx context.Context, friendshipID primitive.ObjectID) (*models.Friendship, error)
	GetAllAcceptedFunc          func(ctx context.Context, limit int64) ([]models.Friendship, error)
	SearchFriendsFunc           func(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error)
	UpdateStatusWithVersionFunc func(ctx context.Context, friendshipID primitive.ObjectID, expectedVersion int64, status models.FriendshipStatus) error
}

func (m *FriendshipRepository) CreateRequest(ctx context.Context, requesterID, receiverID primitive.ObjectID) (*models.Friendship, error) {
	if m.CreateRequestFunc != nil {
		return m.CreateRequestFunc(ctx, requesterID, receiverID)
	}
	return nil, nil
}

func (m *FriendshipRepository) UpdateStatus(ctx context.Context, friendshipID, receiverID primitive.ObjectID, status models.FriendshipStatus) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, friendshipID, receiverID, status)
	}
	return nil
}

func (m *FriendshipRepository) FindByUsers(ctx context.Context, user1, user2 primitive.ObjectID) (*models.Friendship, error) {
	if m.FindByUsersFunc != nil {
		return m.FindByUsersFunc(ctx, user1, user2)
	}
	return nil, nil
}

func (m *FriendshipRepository) AreFriends(ctx context.Context, user1, user2 primitive.ObjectID) (bool, error) {
	if m.AreFriendsFunc != nil {
		return m.AreFriendsFunc(ctx, user1, user2)
	}
	return false, nil
}

func (m *FriendshipRepository) GetFriendships(ctx context.Context, userID primitive.ObjectID, status models.FriendshipStatus, page, limit int64) ([]models.PopulatedFriendship, int64, error) {
	if m.GetFriendshipsFunc != nil {
		return m.GetFriendshipsFunc(ctx, userID, status, page, limit)
	}
	return nil, 0, nil
}

func (m *FriendshipRepository) DeleteFriendship(ctx context.Context, user1, user2 primitive.ObjectID) error {
	if m.DeleteFriendshipFunc != nil {
		return m.DeleteFriendshipFunc(ctx, user1, user2)
	}
	return nil
}

func (m *FriendshipRepository) GetByID(ctx context.Context, friendshipID primitive.ObjectID) (*models.Friendship, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, friendshipID)
	}
	return nil, nil
}

func (m *FriendshipRepository) GetAllAccepted(ctx context.Context, limit int64) ([]models.Friendship, error) {
	if m.GetAllAcceptedFunc != nil {
		return m.GetAllAcceptedFunc(ctx, limit)
	}
	return nil, nil
}

func (m *FriendshipRepository) SearchFriends(ctx context.Context, userID primitive.ObjectID, query string, limit int64) ([]models.UserShortResponse, error) {
	if m.SearchFriendsFunc != nil {
		return m.SearchFriendsFunc(ctx, userID, query, limit)
	}
	return nil, nil
}
