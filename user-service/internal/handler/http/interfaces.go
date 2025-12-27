package http

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	Register(ctx context.Context, user *models.User) (*models.AuthResponse, error)
	Login(ctx context.Context, email, password string) (*models.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error)
}

// UserService defines the interface for user management operations
type UserService interface {
	GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	UpdateProfileFields(ctx context.Context, userID primitive.ObjectID, fullName, bio, avatar, coverPhoto, location, website string) (*models.User, error)
	UpdateEmail(ctx context.Context, userID primitive.ObjectID, email string) error
	UpdatePassword(ctx context.Context, userID primitive.ObjectID, currentPassword, newPassword string) error
	UpdatePrivacySettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdatePrivacySettingsRequest) error
	UpdateNotificationSettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdateNotificationSettingsRequest) error
	ToggleTwoFactor(ctx context.Context, userID primitive.ObjectID, enable bool) error
	DeactivateAccount(ctx context.Context, userID primitive.ObjectID) error
	GetUserStatus(ctx context.Context, userIDStr string) (string, int64, error)
}
