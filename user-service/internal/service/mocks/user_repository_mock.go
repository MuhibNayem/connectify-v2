package mocks

import (
	"context"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	FindUserByIDFunc    func(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	FindUserByEmailFunc func(ctx context.Context, email string) (*models.User, error)
	FindUsersByIDsFunc  func(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error)
	UpdateUserFunc      func(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error)
	FindUsersFunc       func(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.User, error)
	CountUsersFunc      func(ctx context.Context, filter bson.M) (int64, error)
	CreateUserFunc      func(ctx context.Context, user *models.User) (*models.User, error)

	// Track calls for verification
	FindUserByIDCalls   []primitive.ObjectID
	FindUsersByIDsCalls [][]primitive.ObjectID
	UpdateUserCalls     []UpdateUserCall
}

type UpdateUserCall struct {
	ID     primitive.ObjectID
	Update bson.M
}

func (m *MockUserRepository) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	m.FindUserByIDCalls = append(m.FindUserByIDCalls, id)
	if m.FindUserByIDFunc != nil {
		return m.FindUserByIDFunc(ctx, id)
	}
	return &models.User{ID: id, Email: "test@example.com", Username: "testuser"}, nil
}

func (m *MockUserRepository) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.FindUserByEmailFunc != nil {
		return m.FindUserByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) FindUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error) {
	m.FindUsersByIDsCalls = append(m.FindUsersByIDsCalls, ids)
	if m.FindUsersByIDsFunc != nil {
		return m.FindUsersByIDsFunc(ctx, ids)
	}
	users := make([]models.User, len(ids))
	for i, id := range ids {
		users[i] = models.User{ID: id, Email: "user@example.com", Username: "user"}
	}
	return users, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
	m.UpdateUserCalls = append(m.UpdateUserCalls, UpdateUserCall{ID: id, Update: update})
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, id, update)
	}
	return &models.User{ID: id, Email: "updated@example.com", UpdatedAt: time.Now()}, nil
}

func (m *MockUserRepository) FindUsers(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.User, error) {
	if m.FindUsersFunc != nil {
		return m.FindUsersFunc(ctx, filter, opts)
	}
	return []models.User{}, nil
}

func (m *MockUserRepository) CountUsers(ctx context.Context, filter bson.M) (int64, error) {
	if m.CountUsersFunc != nil {
		return m.CountUsersFunc(ctx, filter)
	}
	return 0, nil
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user)
	}
	user.ID = primitive.NewObjectID()
	return user, nil
}

func (m *MockUserRepository) FindUserByUserName(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}

func (m *MockUserRepository) AddFriend(ctx context.Context, userID1, userID2 primitive.ObjectID) error {
	return nil
}

func (m *MockUserRepository) RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error {
	return nil
}
