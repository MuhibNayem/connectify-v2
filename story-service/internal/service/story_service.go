package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	userpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/validation"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoryRepository interface {
	CreateStory(ctx context.Context, story *models.Story) (*models.Story, error)
	GetStoryByID(ctx context.Context, id primitive.ObjectID) (*models.Story, error)
	DeleteStory(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error
	GetActiveStoryAuthors(ctx context.Context, viewerID primitive.ObjectID, userIDs []primitive.ObjectID, limit, offset int) ([]primitive.ObjectID, error)
	GetStoriesForUsers(ctx context.Context, viewerID primitive.ObjectID, authorIDs []primitive.ObjectID) ([]models.Story, error)
	GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error)
	AddViewer(ctx context.Context, storyID primitive.ObjectID, viewerID primitive.ObjectID) error
	AddReaction(ctx context.Context, storyID primitive.ObjectID, reaction models.StoryReaction) error
	GetStoryViewersWithReactions(ctx context.Context, storyID primitive.ObjectID) ([]models.StoryViewerResponse, error)
}

type StoryService struct {
	storyRepo   StoryRepository
	broadcaster producer.StoryBroadcaster
	userClient  userpb.UserServiceClient
	breaker     *resilience.CircuitBreaker
	metrics     *metrics.BusinessMetrics
	logger      *slog.Logger
}

func NewStoryService(
	storyRepo StoryRepository,
	broadcaster producer.StoryBroadcaster,
	userClient userpb.UserServiceClient,
	breaker *resilience.CircuitBreaker,
	metrics *metrics.BusinessMetrics,
	logger *slog.Logger,
) *StoryService {
	if logger == nil {
		logger = slog.Default()
	}
	return &StoryService{
		storyRepo:   storyRepo,
		broadcaster: broadcaster,
		userClient:  userClient,
		breaker:     breaker,
		metrics:     metrics,
		logger:      logger,
	}
}

func (s *StoryService) CreateStory(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor, req CreateStoryRequest) (*models.Story, error) {
	if err := validation.ValidateCreateStoryRequest(req.MediaURL, req.MediaType, req.Privacy, req.AllowedViewers, req.BlockedViewers); err != nil {
		return nil, err
	}

	privacy := req.Privacy
	if privacy == "" {
		privacy = models.PrivacySettingFriends
	}

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
		MediaURL:       validation.SanitizeString(req.MediaURL),
		MediaType:      validation.SanitizeString(req.MediaType),
		Privacy:        privacy,
		AllowedViewers: allowedViewers,
		BlockedViewers: blockedViewers,
	}

	createdStory, err := s.storyRepo.CreateStory(ctx, story)
	if err != nil {
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.IncrementStoriesCreated()
	}

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

func (s *StoryService) GetStory(ctx context.Context, storyID, viewerID primitive.ObjectID) (*models.Story, error) {
	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return nil, errors.New("story not found")
	}

	if !s.canViewStory(ctx, story, viewerID) {
		return nil, errors.New("story not found")
	}

	return story, nil
}

func (s *StoryService) canViewStory(ctx context.Context, story *models.Story, viewerID primitive.ObjectID) bool {
	// 1. Owner always has access
	if story.UserID == viewerID {
		return true
	}

	// 2. Fetch relationship status (scalable check via RPC)
	rel, err := s.getRelationship(ctx, viewerID, story.UserID)
	if err != nil {
		s.logger.Warn("Failed to check relationship", "error", err)
		// Fail safe (deny) if relationship check fails, unless it's Public (debatable, but safer)
		// For Public, maybe we allow if error? No, safer to deny if we can't check blocks.
		return false
	}

	// Global Block Check: If viewer is blocked by author, they can't see ANYTHING
	if rel.IsBlockedByTarget {
		return false
	}

	switch story.Privacy {
	case models.PrivacySettingPublic:
		return true

	case models.PrivacySettingFriends:
		return rel.IsFriend

	case models.PrivacySettingCustom:
		// Check if viewer is in the AllowedViewers list
		for _, allowedID := range story.AllowedViewers {
			if allowedID == viewerID {
				return true
			}
		}
		return false

	case models.PrivacySettingFriendsExcept:
		// Must be a friend AND not blocked
		if !rel.IsFriend {
			return false
		}
		for _, blockedID := range story.BlockedViewers {
			if blockedID == viewerID {
				return false
			}
		}
		return true

	case models.PrivacySettingOnlyMe:
		return false

	default:
		return false
	}
}

func (s *StoryService) getRelationship(ctx context.Context, userID, targetID primitive.ObjectID) (*userpb.CheckRelationshipResponse, error) {
	if s.userClient == nil || s.breaker == nil {
		return nil, errors.New("user service unavailable")
	}

	result, err := s.breaker.Execute(ctx, func() (interface{}, error) {
		return s.userClient.CheckRelationship(ctx, &userpb.CheckRelationshipRequest{
			UserId:   userID.Hex(),
			TargetId: targetID.Hex(),
		})
	})

	if err != nil {
		return nil, err
	}
	return result.(*userpb.CheckRelationshipResponse), nil
}

// Deprecated: Use getRelationship instead
func (s *StoryService) isUserFriend(ctx context.Context, userID, friendID primitive.ObjectID) bool {
	rel, err := s.getRelationship(ctx, userID, friendID)
	if err != nil {
		return false
	}
	return rel.IsFriend
}

func (s *StoryService) DeleteStory(ctx context.Context, storyID, userID primitive.ObjectID) error {
	err := s.storyRepo.DeleteStory(ctx, storyID, userID)
	if err != nil {
		return err
	}

	if s.metrics != nil {
		s.metrics.IncrementStoriesDeleted()
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishStoryDeleted(ctx, producer.StoryDeletedEvent{
			StoryID: storyID.Hex(),
			UserID:  userID.Hex(),
		})
	}

	return nil
}

func (s *StoryService) GetStoriesFeed(ctx context.Context, viewerID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int) ([]models.Story, error) {
	if s.metrics != nil {
		s.metrics.IncrementFeedRequests()
	}

	userIDs := make([]primitive.ObjectID, len(friendIDs)+1)
	userIDs[0] = viewerID
	copy(userIDs[1:], friendIDs)

	authorIDs, err := s.storyRepo.GetActiveStoryAuthors(ctx, viewerID, userIDs, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(authorIDs) == 0 {
		return []models.Story{}, nil
	}

	return s.storyRepo.GetStoriesForUsers(ctx, viewerID, authorIDs)
}

func (s *StoryService) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	return s.storyRepo.GetUserStories(ctx, userID)
}

func (s *StoryService) RecordView(ctx context.Context, storyID, viewerID primitive.ObjectID) error {
	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return errors.New("story not found")
	}

	if !s.canViewStory(ctx, story, viewerID) {
		return errors.New("story not found")
	}

	if story.UserID == viewerID {
		return nil
	}

	if s.metrics != nil {
		s.metrics.IncrementStoriesViewed()
	}

	err = s.storyRepo.AddViewer(ctx, storyID, viewerID)
	if err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishStoryViewed(ctx, producer.StoryViewedEvent{
			StoryID:  storyID.Hex(),
			OwnerID:  story.UserID.Hex(),
			ViewerID: viewerID.Hex(),
			ViewedAt: time.Now(),
		})
	}

	return nil
}

func (s *StoryService) ReactToStory(ctx context.Context, storyID, userID primitive.ObjectID, reactionType string) error {
	if err := validation.ValidateReactionType(reactionType); err != nil {
		return err
	}

	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return errors.New("story not found")
	}

	if !s.canViewStory(ctx, story, userID) {
		return errors.New("story not found")
	}

	reaction := models.StoryReaction{
		StoryID:   storyID,
		UserID:    userID,
		Type:      reactionType,
		CreatedAt: time.Now(),
	}

	err = s.storyRepo.AddReaction(ctx, storyID, reaction)
	if err != nil {
		return err
	}

	if s.metrics != nil {
		s.metrics.IncrementReactions()
	}

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

func (s *StoryService) GetStoryViewers(ctx context.Context, storyID, userID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	story, err := s.storyRepo.GetStoryByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if story.UserID != userID {
		return nil, errors.New("unauthorized: only author can view viewers")
	}

	if s.metrics != nil {
		s.metrics.IncrementViewersAccessed()
	}

	return s.storyRepo.GetStoryViewersWithReactions(ctx, storyID)
}

type CreateStoryRequest struct {
	MediaURL       string
	MediaType      string
	Privacy        models.PrivacySettingType
	AllowedViewers []string
	BlockedViewers []string
}
