package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/MuhibNayem/connectify-v2/reel-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	userpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/sync/singleflight"
)

type ReelRepository interface {
	CreateReel(ctx context.Context, reel *models.Reel) (*models.Reel, error)
	GetReelByID(ctx context.Context, id primitive.ObjectID) (*models.Reel, error)
	GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error)
	DeleteReel(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error
	IncrementViews(ctx context.Context, id primitive.ObjectID) error
	GetReelsFeed(ctx context.Context, userID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int64) ([]models.Reel, error)
	AddComment(ctx context.Context, reelID primitive.ObjectID, comment models.Comment) error
	GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error)
	AddReply(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, reply models.Reply) error
	GetReaction(ctx context.Context, targetID primitive.ObjectID, userID primitive.ObjectID) (*models.Reaction, error)
	AddReaction(ctx context.Context, reaction *models.Reaction) error
	RemoveReaction(ctx context.Context, reaction *models.Reaction) error
	ReactToComment(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, userID primitive.ObjectID, reactionType models.ReactionType) error
}

type ReelService struct {
	reelRepo     ReelRepository
	broadcaster  producer.ReelBroadcaster
	userClient   userpb.UserServiceClient
	breaker      *resilience.CircuitBreaker
	metrics      *metrics.BusinessMetrics
	logger       *slog.Logger
	redisClient  *redis.ClusterClient
	requestGroup singleflight.Group
}

func NewReelService(
	reelRepo ReelRepository,
	broadcaster producer.ReelBroadcaster,
	userClient userpb.UserServiceClient,
	breaker *resilience.CircuitBreaker,
	metrics *metrics.BusinessMetrics,
	logger *slog.Logger,
	redisClient *redis.ClusterClient,
) *ReelService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReelService{
		reelRepo:    reelRepo,
		broadcaster: broadcaster,
		userClient:  userClient,
		breaker:     breaker,
		metrics:     metrics,
		logger:      logger,
		redisClient: redisClient,
	}
}

func (s *ReelService) CreateReel(ctx context.Context, userID primitive.ObjectID, req CreateReelRequest) (*models.Reel, error) {
	// Fetch author info from user-service
	author, err := s.resolveAuthor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolving author: %w", err)
	}

	if req.VideoURL == "" {
		return nil, fmt.Errorf("video URL is required")
	}

	privacy := req.Privacy
	if privacy == "" {
		privacy = models.PrivacySettingPublic
	}

	reel := &models.Reel{
		UserID:         userID,
		VideoURL:       req.VideoURL,
		ThumbnailURL:   req.ThumbnailURL,
		Caption:        req.Caption,
		Duration:       req.Duration,
		Privacy:        privacy,
		AllowedViewers: req.AllowedViewers,
		BlockedViewers: req.BlockedViewers,
		Author:         author,
	}

	createdReel, err := s.reelRepo.CreateReel(ctx, reel)
	if err != nil {
		return nil, err
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishReelCreated(ctx, producer.ReelCreatedEvent{
			ReelID:   createdReel.ID.Hex(),
			UserID:   userID.Hex(),
			VideoURL: req.VideoURL,
		})
	}

	if s.metrics != nil {
		s.metrics.ReelsCreated.Inc()
	}

	s.logger.Info("Reel created", "reel_id", createdReel.ID.Hex(), "user_id", userID.Hex())

	return createdReel, nil
}

// GetReelsFeed returns a list of reels for the user's feed with optimized DB filtering
func (s *ReelService) GetReelsFeed(ctx context.Context, userID primitive.ObjectID, limit, offset int64) ([]models.Reel, error) {
	// Get user's friends to filter content
	friendIDs, err := s.getFriendIDs(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get friend IDs, returning public reels only", "error", err, "user_id", userID.Hex())
		friendIDs = []primitive.ObjectID{}
	}

	// Fetch Reels with DB-level filtering
	return s.reelRepo.GetReelsFeed(ctx, userID, friendIDs, limit, offset)
}

// getFriendIDs fetches friend IDs via gRPC to user-service with Redis caching
func (s *ReelService) getFriendIDs(ctx context.Context, userID primitive.ObjectID) ([]primitive.ObjectID, error) {
	if s.userClient == nil {
		return nil, fmt.Errorf("user service client not available")
	}

	cacheKey := fmt.Sprintf("reel:friends:%s", userID.Hex())

	// Check Redis cache first
	if s.redisClient != nil {
		if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil && cached != "" {
			// Parse cached friend IDs
			friendIDs := parseFriendIDsFromCache(cached)
			if len(friendIDs) > 0 {
				return friendIDs, nil
			}
		}
	}

	// Call user-service via gRPC with circuit breaker
	var resp *userpb.GetFriendIDsResponse
	var err error

	if s.breaker != nil {
		result, cbErr := s.breaker.Execute(ctx, func() (interface{}, error) {
			return s.userClient.GetFriendIDs(ctx, &userpb.GetFriendIDsRequest{UserId: userID.Hex()})
		})
		if cbErr != nil {
			return nil, cbErr
		}
		resp = result.(*userpb.GetFriendIDsResponse)
	} else {
		resp, err = s.userClient.GetFriendIDs(ctx, &userpb.GetFriendIDsRequest{UserId: userID.Hex()})
		if err != nil {
			return nil, err
		}
	}

	// Convert string IDs to ObjectIDs
	friendIDs := make([]primitive.ObjectID, 0, len(resp.FriendIds))
	for _, id := range resp.FriendIds {
		if oid, err := primitive.ObjectIDFromHex(id); err == nil {
			friendIDs = append(friendIDs, oid)
		}
	}

	// Cache in Redis (5 minute TTL)
	if s.redisClient != nil && len(resp.FriendIds) > 0 {
		cacheValue := stringSliceToCache(resp.FriendIds)
		s.redisClient.Set(ctx, cacheKey, cacheValue, 5*time.Minute)
	}

	return friendIDs, nil
}

func (s *ReelService) GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error) {
	return s.reelRepo.GetUserReels(ctx, userID)
}

func (s *ReelService) GetReel(ctx context.Context, reelID primitive.ObjectID) (*models.Reel, error) {
	return s.reelRepo.GetReelByID(ctx, reelID)
}

func (s *ReelService) DeleteReel(ctx context.Context, reelID, userID primitive.ObjectID) error {
	if err := s.reelRepo.DeleteReel(ctx, reelID, userID); err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishReelDeleted(ctx, producer.ReelDeletedEvent{
			ReelID: reelID.Hex(),
			UserID: userID.Hex(),
		})
	}

	s.logger.Info("Reel deleted", "reel_id", reelID.Hex(), "user_id", userID.Hex())

	return nil
}

// IncrementViews publishes a view event for async batch processing
// For TikTok-scale (1M+ views/sec), we avoid direct DB writes here.
// A separate Kafka consumer batches updates (e.g., $inc: { views: 500 }) to MongoDB.
func (s *ReelService) IncrementViews(ctx context.Context, reelID, viewerID primitive.ObjectID) error {
	// Fetch reel to check author
	reel, err := s.reelRepo.GetReelByID(ctx, reelID)
	if err != nil {
		return err
	}

	// Don't count self-views
	if reel.UserID == viewerID {
		return nil
	}

	// Only publish Kafka event - consumer handles batch DB updates
	if s.broadcaster != nil {
		s.broadcaster.PublishReelViewed(ctx, producer.ReelViewedEvent{
			ReelID:   reelID.Hex(),
			ViewerID: viewerID.Hex(),
		})
	}

	// Note: Direct DB write removed for scale optimization
	// View counts are updated by the Kafka consumer in batches
	return nil
}

// ReactToReel handles toggling a reaction on a reel
func (s *ReelService) ReactToReel(ctx context.Context, reelID, userID primitive.ObjectID, reactionType models.ReactionType) error {
	// Check for existing reaction
	existingReaction, err := s.reelRepo.GetReaction(ctx, reelID, userID)
	if err != nil {
		return err
	}

	if existingReaction != nil {
		// Already reacted
		if existingReaction.Type == reactionType {
			// Same type -> Remove (Toggle OFF)
			return s.reelRepo.RemoveReaction(ctx, existingReaction)
		}

		// Different type -> Remove old, Add new (Change reaction)
		err = s.reelRepo.RemoveReaction(ctx, existingReaction)
		if err != nil {
			return err
		}
	}

	// Add new reaction
	newReaction := &models.Reaction{
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		TargetID:   reelID,
		TargetType: "reel",
		Type:       reactionType,
		CreatedAt:  time.Now(),
	}

	return s.reelRepo.AddReaction(ctx, newReaction)
}

// AddComment adds a comment to a reel
func (s *ReelService) AddComment(ctx context.Context, reelID, userID primitive.ObjectID, content string, explicitMentions []primitive.ObjectID) (*models.Comment, error) {
	author, err := s.resolveAuthor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolving author: %w", err)
	}

	// Parse mentions from content and merge with explicit mentions
	parsedMentions := s.ParseMentions(ctx, content)

	// Deduplicate mentions
	mentionMap := make(map[string]primitive.ObjectID)
	for _, id := range parsedMentions {
		mentionMap[id.Hex()] = id
	}
	for _, id := range explicitMentions {
		mentionMap[id.Hex()] = id
	}

	var finalMentions []primitive.ObjectID
	for _, id := range mentionMap {
		finalMentions = append(finalMentions, id)
	}

	comment := models.Comment{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Content:   content,
		Author:    author,
		Mentions:  finalMentions,
		CreatedAt: time.Now(),
	}

	err = s.reelRepo.AddComment(ctx, reelID, comment)
	if err != nil {
		return nil, err
	}

	if s.metrics != nil {
		s.metrics.CommentsAdded.Inc()
	}

	return &comment, nil
}

// GetComments retrieves all comments for a reel with pagination
func (s *ReelService) GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.reelRepo.GetComments(ctx, reelID, limit, offset)
}

// AddReply adds a reply to a comment on a reel
func (s *ReelService) AddReply(ctx context.Context, reelID, commentID, userID primitive.ObjectID, content string) (*models.Reply, error) {
	author, err := s.resolveAuthor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolving author: %w", err)
	}

	// Parse mentions from content
	mentions := s.ParseMentions(ctx, content)

	reply := models.Reply{
		ID:        primitive.NewObjectID(),
		CommentID: commentID,
		UserID:    userID,
		Author:    author,
		Mentions:  mentions,
		Content:   content,
		CreatedAt: time.Now(),
	}

	err = s.reelRepo.AddReply(ctx, reelID, commentID, reply)
	if err != nil {
		return nil, err
	}

	return &reply, nil
}

// ReactToComment toggles a reaction on a comment
func (s *ReelService) ReactToComment(ctx context.Context, reelID, commentID, userID primitive.ObjectID, reactionType models.ReactionType) error {
	return s.reelRepo.ReactToComment(ctx, reelID, commentID, userID, reactionType)
}

// ParseMentions extracts @username mentions from content and validates them via user-service
// Uses batch RPC (GetUsersByUsernames) for efficient single network call
func (s *ReelService) ParseMentions(ctx context.Context, content string) []primitive.ObjectID {
	// Regex for @username (alphanumeric + underscores, min 3 chars)
	re := regexp.MustCompile(`@([a-zA-Z0-9_]{3,})`)
	matches := re.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return []primitive.ObjectID{}
	}

	// Collect unique usernames
	uniqueUsernames := make([]string, 0)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			uniqueUsernames = append(uniqueUsernames, match[1])
		}
	}

	if len(uniqueUsernames) == 0 || s.userClient == nil {
		return []primitive.ObjectID{}
	}

	// Single batch RPC call to resolve all usernames
	resp, err := s.userClient.GetUsersByUsernames(ctx, &userpb.GetUsersByUsernamesRequest{
		Usernames: uniqueUsernames,
	})
	if err != nil {
		s.logger.Warn("Failed to resolve mentions via batch RPC", "error", err)
		return []primitive.ObjectID{}
	}

	// Convert to ObjectIDs
	mentionIDs := make([]primitive.ObjectID, 0, len(resp.Users))
	for _, user := range resp.Users {
		if oid, err := primitive.ObjectIDFromHex(user.Id); err == nil {
			mentionIDs = append(mentionIDs, oid)
		}
	}

	return mentionIDs
}

// Helper functions for cache serialization
func parseFriendIDsFromCache(cached string) []primitive.ObjectID {
	if cached == "" {
		return nil
	}
	// Simple comma-separated format
	ids := make([]primitive.ObjectID, 0)
	for _, idStr := range splitString(cached, ",") {
		if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
			ids = append(ids, oid)
		}
	}
	return ids
}

func stringSliceToCache(ids []string) string {
	result := ""
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += id
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	current := ""
	for _, c := range s {
		if string(c) == sep {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func (s *ReelService) resolveAuthor(ctx context.Context, userID primitive.ObjectID) (models.PostAuthor, error) {
	if s.userClient == nil {
		// Fallback for tests or when service is isolated
		return models.PostAuthor{ID: userID.Hex()}, nil
	}

	// Use singleflight to deduplicate concurrent requests for the same user
	val, err, _ := s.requestGroup.Do(userID.Hex(), func() (interface{}, error) {
		// Check cache? For now just call user service (it has caching)
		resp, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID.Hex()})
		if err != nil {
			return models.PostAuthor{}, err
		}

		user := resp.GetUser()
		return models.PostAuthor{
			ID:       user.Id,
			Username: user.Username,
			Avatar:   user.Avatar,
			FullName: user.FullName,
		}, nil
	})

	if err != nil {
		return models.PostAuthor{}, err
	}
	return val.(models.PostAuthor), nil
}

type CreateReelRequest struct {
	VideoURL       string                    `json:"video_url"`
	ThumbnailURL   string                    `json:"thumbnail_url"`
	Caption        string                    `json:"caption"`
	Duration       int                       `json:"duration"`
	Privacy        models.PrivacySettingType `json:"privacy"`
	AllowedViewers []primitive.ObjectID      `json:"allowed_viewers"`
	BlockedViewers []primitive.ObjectID      `json:"blocked_viewers"`
}
