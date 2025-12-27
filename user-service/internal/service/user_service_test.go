package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"user-service/config"
	"user-service/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

func TestUserService_GetUserByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    primitive.ObjectID
		mockSetup func(*mocks.MockUserRepository)
		wantErr   bool
		wantUser  bool
	}{
		{
			name:   "successfully get user by ID",
			userID: primitive.NewObjectID(),
			mockSetup: func(repo *mocks.MockUserRepository) {
				repo.FindUserByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
					return &models.User{ID: id, Email: "test@example.com", Username: "testuser"}, nil
				}
			},
			wantErr:  false,
			wantUser: true,
		},
		{
			name:   "user not found",
			userID: primitive.NewObjectID(),
			mockSetup: func(repo *mocks.MockUserRepository) {
				repo.FindUserByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
					return nil, mongo.ErrNoDocuments
				}
			},
			wantErr:  true,
			wantUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockUserRepository{}
			tt.mockSetup(mockRepo)

			svc := newTestUserService(mockRepo, nil, nil)
			user, err := svc.GetUserByID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantUser {
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
			}
		})
	}
}

func TestUserService_GetUsersByIDs(t *testing.T) {
	ids := []primitive.ObjectID{
		primitive.NewObjectID(),
		primitive.NewObjectID(),
		primitive.NewObjectID(),
	}

	tests := []struct {
		name      string
		ids       []primitive.ObjectID
		mockSetup func(*mocks.MockUserRepository)
		wantCount int
		wantErr   bool
	}{
		{
			name: "batch fetch multiple users",
			ids:  ids,
			mockSetup: func(repo *mocks.MockUserRepository) {
				repo.FindUsersByIDsFunc = func(ctx context.Context, reqIDs []primitive.ObjectID) ([]models.User, error) {
					users := make([]models.User, len(reqIDs))
					for i, id := range reqIDs {
						users[i] = models.User{ID: id, Username: "user"}
					}
					return users, nil
				}
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "empty IDs returns empty slice",
			ids:  []primitive.ObjectID{},
			mockSetup: func(repo *mocks.MockUserRepository) {
				// Should not call repo
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "database error",
			ids:  ids,
			mockSetup: func(repo *mocks.MockUserRepository) {
				repo.FindUsersByIDsFunc = func(ctx context.Context, reqIDs []primitive.ObjectID) ([]models.User, error) {
					return nil, errors.New("database error")
				}
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockUserRepository{}
			tt.mockSetup(mockRepo)

			svc := newTestUserService(mockRepo, nil, nil)
			users, err := svc.GetUsersByIDs(context.Background(), tt.ids)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, users, tt.wantCount)
			}

			// Verify batch call was made (not N+1)
			if len(tt.ids) > 0 && !tt.wantErr {
				assert.Equal(t, 1, len(mockRepo.FindUsersByIDsCalls), "should make single batch call")
			}
		})
	}
}

func TestUserService_UpdateEmail(t *testing.T) {
	userID := primitive.NewObjectID()

	tests := []struct {
		name      string
		userID    primitive.ObjectID
		newEmail  string
		mockSetup func(*mocks.MockUserRepository, *mocks.MockEventProducer)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successfully update email",
			userID:   userID,
			newEmail: "newemail@example.com",
			mockSetup: func(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer) {
				repo.UpdateUserFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
					return &models.User{ID: id, Email: update["email"].(string)}, nil
				}
			},
			wantErr: false,
		},
		{
			name:     "email already in use (duplicate key)",
			userID:   userID,
			newEmail: "existing@example.com",
			mockSetup: func(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer) {
				repo.UpdateUserFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
					// Simulate duplicate key error
					return nil, mongo.WriteException{
						WriteErrors: []mongo.WriteError{{Code: 11000}},
					}
				}
			},
			wantErr: true,
			errMsg:  "email already in use",
		},
		{
			name:     "database error",
			userID:   userID,
			newEmail: "test@example.com",
			mockSetup: func(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer) {
				repo.UpdateUserFunc = func(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error) {
					return nil, errors.New("connection failed")
				}
			},
			wantErr: true,
			errMsg:  "failed to update email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockUserRepository{}
			mockProducer := &mocks.MockEventProducer{}
			tt.mockSetup(mockRepo, mockProducer)

			svc := newTestUserService(mockRepo, mockProducer, nil)
			err := svc.UpdateEmail(context.Background(), tt.userID, tt.newEmail)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify event was published
				assert.Len(t, mockProducer.ProduceCalls, 1, "should publish USER_UPDATED event")
			}
		})
	}
}

func TestUserService_UpdatePassword(t *testing.T) {
	userID := primitive.NewObjectID()
	// bcrypt hash of "OldPass123"
	hashedPassword := "$2a$10$N9qo8uLOickgx2ZMRZoMy.Mrq6Z8.Jxk4Rq5W1Y5aSz1YKDJ6YKSO"

	tests := []struct {
		name            string
		userID          primitive.ObjectID
		currentPassword string
		newPassword     string
		mockSetup       func(*mocks.MockUserRepository, *mocks.MockEventProducer)
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "user not found",
			userID:          userID,
			currentPassword: "OldPass123",
			newPassword:     "NewPass456",
			mockSetup: func(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer) {
				repo.FindUserByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
					return nil, errors.New("user not found")
				}
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:            "incorrect current password",
			userID:          userID,
			currentPassword: "WrongPassword",
			newPassword:     "NewPass456",
			mockSetup: func(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer) {
				repo.FindUserByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
					return &models.User{ID: id, Password: hashedPassword}, nil
				}
			},
			wantErr: true,
			errMsg:  "current password is incorrect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockUserRepository{}
			mockProducer := &mocks.MockEventProducer{}
			tt.mockSetup(mockRepo, mockProducer)

			svc := newTestUserService(mockRepo, mockProducer, nil)
			err := svc.UpdatePassword(context.Background(), tt.userID, tt.currentPassword, tt.newPassword)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper to create test UserService with mocks
func newTestUserService(repo *mocks.MockUserRepository, producer *mocks.MockEventProducer, redisMock *mocks.MockRedisClient) *UserService {
	return &UserService{
		userRepo:    repo,
		producer:    producer,
		redisClient: nil, // Redis tests can be added separately
		cfg:         &config.Config{},
		logger:      slog.Default(),
		metrics:     nil, // Metrics tests can be added separately
	}
}
