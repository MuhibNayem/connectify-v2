package httpapi

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StoryService defines the interface for story operations
type StoryService interface {
	CreateStory(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor, req service.CreateStoryRequest) (*models.Story, error)
	GetStory(ctx context.Context, storyID, viewerID primitive.ObjectID) (*models.Story, error)
	DeleteStory(ctx context.Context, storyID, userID primitive.ObjectID) error
	GetStoriesFeed(ctx context.Context, viewerID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int) ([]models.Story, error)
	GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error)
	RecordView(ctx context.Context, storyID, viewerID primitive.ObjectID) error
	ReactToStory(ctx context.Context, storyID, userID primitive.ObjectID, reactionType string) error
	GetStoryViewers(ctx context.Context, storyID, userID primitive.ObjectID) ([]models.StoryViewerResponse, error)
}
