package service

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	FindUserByUserName(ctx context.Context, username string) (*models.User, error)
	FindUsersByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.User, error)
	FindUsersByUsernames(ctx context.Context, usernames []string) ([]models.User, error)
	FindUsers(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.User, error)
	CountUsers(ctx context.Context, filter bson.M) (int64, error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	UpdateUser(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.User, error)
	AddFriend(ctx context.Context, userID1, userID2 primitive.ObjectID) error
	RemoveFriend(ctx context.Context, userID, friendID primitive.ObjectID) error
}

// EventProducer defines the interface for Kafka event publishing
type EventProducer interface {
	Produce(ctx context.Context, key, value []byte) error
	Close() error
}
