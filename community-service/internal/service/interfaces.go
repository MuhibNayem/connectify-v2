package service

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CommunityRepository defines the interface for community persistence
type CommunityRepository interface {
	Create(ctx context.Context, community *models.Community) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Community, error)
	Update(ctx context.Context, community *models.Community) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, limit, page int64) ([]models.Community, int64, error)
	Search(ctx context.Context, query string, limit, page int64) ([]models.Community, int64, error)

	AddMember(ctx context.Context, communityID, userID primitive.ObjectID) error
	RemoveMember(ctx context.Context, communityID, userID primitive.ObjectID) error

	AddPendingMember(ctx context.Context, communityID, userID primitive.ObjectID) error
	RemovePendingMember(ctx context.Context, communityID, userID primitive.ObjectID) error

	GetUserCommunities(ctx context.Context, userID primitive.ObjectID) ([]models.Community, error)
	GetMembers(ctx context.Context, communityID primitive.ObjectID, limit, page int64) ([]models.User, int64, error)
	GetAdmins(ctx context.Context, communityID primitive.ObjectID) ([]models.User, error)
	GetPendingMembers(ctx context.Context, communityID primitive.ObjectID, limit, page int64) ([]models.User, int64, error)
}
