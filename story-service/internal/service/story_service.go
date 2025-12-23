package service

import (
	"context"
	"errors"
	"time"

	"gitlab.com/spydotech-group/shared-entity/models"
	"gitlab.com/spydotech-group/story-service/internal/producer"
	"gitlab.com/spydotech-group/story-service/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoryService struct {
	storyRepo   *repository.StoryRepository
	broadcaster producer.StoryBroadcaster
}

func NewStoryService(storyRepo *repository.StoryRepository, broadcaster producer.StoryBroadcaster) *StoryService {
	return &StoryService{
		storyRepo:   storyRepo,
		broadcaster: broadcaster,
	}
}

// CreateStory creates a new story and publishes a real-time event
func (s *StoryService) CreateStory(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor, req CreateStoryRequest) (*models.Story, error) {
	privacy := req.Privacy
	if privacy == "" {
		privacy = models.PrivacySettingFriends
	}

	// Convert allowed/blocked viewers to ObjectIDs
	allowedViewers := make([]primitive.ObjectID, 0)
	for _, id := range req.AllowedViewers {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			allowedViewers = append(allowedViewers, oid)
		}
	}

	blockedViewers := make([]primitive.ObjectID, 0)
	for _, id := range req.BlockedViewers {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			blockedViewers = append(blockedViewers, oid)
		}
	}

	story := &models.Story{
		UserID:         userID,
		Author:         author,
		MediaURL:       req.MediaURL,
		MediaType:      req.MediaType,
		Privacy:        privacy,
		AllowedViewers: allowedViewers,
		BlockedViewers: blockedViewers,
	}

	createdStory, err := s.storyRepo.CreateStory(ctx, story)
	if err != nil {
		return nil, err
	}

	// Publish real-time event
	if s.broadcaster != nil {
		s.broadcaster.PublishStoryCreated(ctx, producer.StoryCreatedEvent{
			StoryID:   createdStory.ID.Hex(),
			UserID:    userID.Hex(),
			Author:    author,
			MediaURL:  createdStory.MediaURL,
			MediaType: createdStory.MediaType,
			CreatedAt: createdStory.CreatedAt,
			ExpiresAt: createdStory.ExpiresAt,
		})
	}

	return createdStory, nil
}

// GetStory retrieves a single story by ID
func (s *StoryService) GetStory(ctx context.Context, storyID primitive.ObjectID) (*models.Story, error) {
	return s.storyRepo.GetStoryByID(ctx, storyID)
}

// DeleteStory deletes a story and publishes a real-time event
func (s *StoryService) DeleteStory(ctx context.Context, storyID, userID primitive.ObjectID) error {
	err := s.storyRepo.DeleteStory(ctx, storyID, userID)
	if err != nil {
		return err
	}

	// Publish real-time event
	if s.broadcaster != nil {
		s.broadcaster.PublishStoryDeleted(ctx, producer.StoryDeletedEvent{
			StoryID: storyID.Hex(),
			UserID:  userID.Hex(),
		})
	}

	return nil
}

// GetStoriesFeed returns paginated stories feed with privacy filtering
func (s *StoryService) GetStoriesFeed(ctx context.Context, viewerID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int) ([]models.Story, error) {
	// Include self in userIDs
	userIDs := make([]primitive.ObjectID, len(friendIDs)+1)
	userIDs[0] = viewerID
	copy(userIDs[1:], friendIDs)

	// Get paginated authors
	authorIDs, err := s.storyRepo.GetActiveStoryAuthors(ctx, viewerID, userIDs, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(authorIDs) == 0 {
		return []models.Story{}, nil
	}

	// Fetch stories for these authors
	return s.storyRepo.GetStoriesForUsers(ctx, viewerID, authorIDs)
}

// GetUserStories returns all active stories for a specific user
func (s *StoryService) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	return s.storyRepo.GetUserStories(ctx, userID)
}

// RecordView records a story view and publishes a real-time event
func (s *StoryService) RecordView(ctx context.Context, storyID, viewerID primitive.ObjectID) error {
	// Fetch story to check author
	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return err
	}

	// Don't count self-views
	if story.UserID == viewerID {
		return nil
	}

	err = s.storyRepo.AddViewer(ctx, storyID, viewerID)
	if err != nil {
		return err
	}

	// Publish real-time event
	if s.broadcaster != nil {
		s.broadcaster.PublishStoryViewed(ctx, producer.StoryViewedEvent{
			StoryID:  storyID.Hex(),
			OwnerID:  story.UserID.Hex(), // Story owner to be notified
			ViewerID: viewerID.Hex(),
			ViewedAt: time.Now(),
		})
	}

	return nil
}

// ReactToStory adds a reaction and publishes a real-time event
func (s *StoryService) ReactToStory(ctx context.Context, storyID, userID primitive.ObjectID, reactionType string) error {
	reaction := models.StoryReaction{
		StoryID:   storyID,
		UserID:    userID,
		Type:      reactionType,
		CreatedAt: time.Now(),
	}

	err := s.storyRepo.AddReaction(ctx, storyID, reaction)
	if err != nil {
		return err
	}

	// Publish real-time event
	if s.broadcaster != nil {
		s.broadcaster.PublishStoryReaction(ctx, producer.StoryReactionEvent{
			StoryID:      storyID.Hex(),
			UserID:       userID.Hex(),
			ReactionType: reactionType,
			CreatedAt:    time.Now(),
		})
	}

	return nil
}

// GetStoryViewers returns viewers with their reactions (only for story owner)
func (s *StoryService) GetStoryViewers(ctx context.Context, storyID, userID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	// Verify ownership
	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if story.UserID != userID {
		return nil, errors.New("unauthorized: only author can view viewers")
	}

	return s.storyRepo.GetStoryViewersWithReactions(ctx, storyID)
}

// Request types
type CreateStoryRequest struct {
	MediaURL       string
	MediaType      string
	Privacy        models.PrivacySettingType
	AllowedViewers []string
	BlockedViewers []string
}
