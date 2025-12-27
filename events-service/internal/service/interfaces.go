package service

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/integration"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventRepository defines the interface for event persistence
type EventRepository interface {
	Create(ctx context.Context, event *models.Event) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error)
	Update(ctx context.Context, event *models.Event) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	List(ctx context.Context, limit, page int64, filter bson.M) ([]models.Event, int64, error)
	AddOrUpdateAttendee(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error
	RemoveAttendee(ctx context.Context, eventID, userID primitive.ObjectID) error
	UpdateStats(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error
	GetUserEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.Event, error)
	GetAttendeesByStatus(ctx context.Context, eventID primitive.ObjectID, status models.RSVPStatus, limit, page int64) ([]models.EventAttendee, int64, error)
	GetCategories(ctx context.Context) ([]models.EventCategory, error)
	IncrementShareCount(ctx context.Context, eventID primitive.ObjectID) error
	AddCoHost(ctx context.Context, eventID primitive.ObjectID, coHost models.EventCoHost) error
	RemoveCoHost(ctx context.Context, eventID, userID primitive.ObjectID) error
	IsCoHost(ctx context.Context, eventID, userID primitive.ObjectID) (bool, error)
	Search(ctx context.Context, query string, filter bson.M, limit, page int64) ([]models.Event, int64, error)
	GetNearbyEvents(ctx context.Context, lat, lng, radiusKm float64, limit, page int64) ([]models.Event, int64, error)
}

// UserRepo defines interface for user interactions
type UserRepo interface {
	FindByID(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error)
	FindByIDs(ctx context.Context, ids []primitive.ObjectID) ([]integration.EventUser, error)
	FindFriendBirthdays(ctx context.Context, friendIDs []primitive.ObjectID) ([]integration.EventUser, []integration.EventUser, error)
	GetFriends(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error)
}

// EventGraphRepo defines interface for graph operations
type EventGraphRepo interface {
	AddAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error
	RemoveAttendee(ctx context.Context, userID, eventID primitive.ObjectID) error
	GetFriendsGoing(ctx context.Context, userID, eventID primitive.ObjectID) ([]string, error)
}

// InvitationRepo defines interface for invitation persistence
type InvitationRepo interface {
	CreateMany(ctx context.Context, invitations []models.EventInvitation) error
	CheckExisting(ctx context.Context, eventID, inviteeID primitive.ObjectID) (*models.EventInvitation, error)
	GetUserInvitations(ctx context.Context, userID primitive.ObjectID, status models.EventInvitationStatus, limit, page int64) ([]models.EventInvitation, int64, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.EventInvitation, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.EventInvitationStatus) error
}

// PostRepo defines interface for event posts
type PostRepo interface {
	Create(ctx context.Context, post *models.EventPost) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.EventPost, error)
	GetByEventID(ctx context.Context, eventID primitive.ObjectID, limit, page int64) ([]models.EventPost, int64, error)
	Update(ctx context.Context, post *models.EventPost) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteByEventID(ctx context.Context, eventID primitive.ObjectID) error
	AddReaction(ctx context.Context, postID primitive.ObjectID, reaction models.EventPostReaction) error
	RemoveReaction(ctx context.Context, postID, userID primitive.ObjectID) error
	GetPostCount(ctx context.Context, eventID primitive.ObjectID) (int64, error)
}
