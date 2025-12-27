package service

import (
	"context"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/reel-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockReelRepository struct {
	mock.Mock
}

func (m *MockReelRepository) CreateReel(ctx context.Context, reel *models.Reel) (*models.Reel, error) {
	args := m.Called(ctx, reel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reel), args.Error(1)
}

func (m *MockReelRepository) GetReelByID(ctx context.Context, id primitive.ObjectID) (*models.Reel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reel), args.Error(1)
}

func (m *MockReelRepository) GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Reel), args.Error(1)
}

func (m *MockReelRepository) DeleteReel(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockReelRepository) IncrementViews(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReelRepository) GetReelsFeed(ctx context.Context, userID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int64) ([]models.Reel, error) {
	args := m.Called(ctx, userID, friendIDs, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Reel), args.Error(1)
}

func (m *MockReelRepository) AddComment(ctx context.Context, reelID primitive.ObjectID, comment models.Comment) error {
	args := m.Called(ctx, reelID, comment)
	return args.Error(0)
}

func (m *MockReelRepository) GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error) {
	args := m.Called(ctx, reelID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Comment), args.Error(1)
}

func (m *MockReelRepository) AddReply(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, reply models.Reply) error {
	args := m.Called(ctx, reelID, commentID, reply)
	return args.Error(0)
}

func (m *MockReelRepository) GetReaction(ctx context.Context, targetID primitive.ObjectID, userID primitive.ObjectID) (*models.Reaction, error) {
	args := m.Called(ctx, targetID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reaction), args.Error(1)
}

func (m *MockReelRepository) AddReaction(ctx context.Context, reaction *models.Reaction) error {
	args := m.Called(ctx, reaction)
	return args.Error(0)
}

func (m *MockReelRepository) RemoveReaction(ctx context.Context, reaction *models.Reaction) error {
	args := m.Called(ctx, reaction)
	return args.Error(0)
}

func (m *MockReelRepository) ReactToComment(ctx context.Context, reelID primitive.ObjectID, commentID primitive.ObjectID, userID primitive.ObjectID, reactionType models.ReactionType) error {
	args := m.Called(ctx, reelID, commentID, userID, reactionType)
	return args.Error(0)
}

type MockBroadcaster struct {
	mock.Mock
}

func (m *MockBroadcaster) PublishReelCreated(ctx context.Context, event producer.ReelCreatedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) PublishReelDeleted(ctx context.Context, event producer.ReelDeletedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) PublishReelViewed(ctx context.Context, event producer.ReelViewedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) Close() error {
	args := m.Called()
	return args.Error(0)
}

func newTestReelService(repo *MockReelRepository, broadcaster *MockBroadcaster) *ReelService {
	breaker := resilience.NewCircuitBreaker(resilience.DefaultConfig("test"), nil)
	return NewReelService(repo, broadcaster, nil, breaker, nil, nil, nil)
}

func TestCreateReel_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	author := models.PostAuthor{
		ID:       userID.Hex(),
		Username: "testuser",
		Avatar:   "avatar.png",
		FullName: "Test User",
	}
	req := CreateReelRequest{
		VideoURL:     "https://example.com/video.mp4",
		ThumbnailURL: "https://example.com/thumb.jpg",
		Caption:      "Test caption",
		Duration:     30,
		Privacy:      models.PrivacySettingPublic,
	}

	expectedReel := &models.Reel{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		VideoURL:     req.VideoURL,
		ThumbnailURL: req.ThumbnailURL,
		Caption:      req.Caption,
		Duration:     req.Duration,
		Privacy:      req.Privacy,
		Author:       author,
	}

	mockRepo.On("CreateReel", ctx, mock.AnythingOfType("*models.Reel")).Return(expectedReel, nil)
	mockBroadcaster.On("PublishReelCreated", ctx, mock.AnythingOfType("producer.ReelCreatedEvent")).Return()

	reel, err := svc.CreateReel(ctx, userID, author, req)

	assert.NoError(t, err)
	assert.NotNil(t, reel)
	assert.Equal(t, req.VideoURL, reel.VideoURL)
	assert.Equal(t, req.Caption, reel.Caption)
	mockRepo.AssertExpectations(t)
	mockBroadcaster.AssertExpectations(t)
}

func TestCreateReel_EmptyVideoURL(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}
	req := CreateReelRequest{
		VideoURL: "",
	}

	reel, err := svc.CreateReel(ctx, userID, author, req)

	assert.Error(t, err)
	assert.Nil(t, reel)
	assert.Contains(t, err.Error(), "video URL is required")
}

func TestCreateReel_DefaultPrivacy(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}
	req := CreateReelRequest{
		VideoURL: "https://example.com/video.mp4",
		Privacy:  "",
	}

	expectedReel := &models.Reel{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		VideoURL: req.VideoURL,
		Privacy:  models.PrivacySettingPublic,
	}

	mockRepo.On("CreateReel", ctx, mock.MatchedBy(func(r *models.Reel) bool {
		return r.Privacy == models.PrivacySettingPublic
	})).Return(expectedReel, nil)
	mockBroadcaster.On("PublishReelCreated", ctx, mock.Anything).Return()

	reel, err := svc.CreateReel(ctx, userID, author, req)

	assert.NoError(t, err)
	assert.Equal(t, models.PrivacySettingPublic, reel.Privacy)
}

// ==================== GetReel Tests ====================

func TestGetReel_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	expectedReel := &models.Reel{
		ID:       reelID,
		Caption:  "Test reel",
		VideoURL: "https://example.com/video.mp4",
	}

	mockRepo.On("GetReelByID", ctx, reelID).Return(expectedReel, nil)

	reel, err := svc.GetReel(ctx, reelID)

	assert.NoError(t, err)
	assert.NotNil(t, reel)
	assert.Equal(t, reelID, reel.ID)
	mockRepo.AssertExpectations(t)
}

// ==================== DeleteReel Tests ====================

func TestDeleteReel_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	mockRepo.On("DeleteReel", ctx, reelID, userID).Return(nil)
	mockBroadcaster.On("PublishReelDeleted", ctx, mock.AnythingOfType("producer.ReelDeletedEvent")).Return()

	err := svc.DeleteReel(ctx, reelID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockBroadcaster.AssertExpectations(t)
}

// ==================== IncrementViews Tests ====================

func TestIncrementViews_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	viewerID := primitive.NewObjectID()
	authorID := primitive.NewObjectID()

	reel := &models.Reel{
		ID:     reelID,
		UserID: authorID,
	}

	mockRepo.On("GetReelByID", ctx, reelID).Return(reel, nil)
	mockBroadcaster.On("PublishReelViewed", ctx, mock.AnythingOfType("producer.ReelViewedEvent")).Return()

	err := svc.IncrementViews(ctx, reelID, viewerID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "IncrementViews")
}

func TestIncrementViews_SelfView_NoIncrement(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	reel := &models.Reel{
		ID:     reelID,
		UserID: userID,
	}

	mockRepo.On("GetReelByID", ctx, reelID).Return(reel, nil)

	err := svc.IncrementViews(ctx, reelID, userID)

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "IncrementViews")
}

// ==================== ReactToReel Tests ====================

func TestReactToReel_NewReaction(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	reactionType := models.ReactionLike

	mockRepo.On("GetReaction", ctx, reelID, userID).Return(nil, nil)
	mockRepo.On("AddReaction", ctx, mock.AnythingOfType("*models.Reaction")).Return(nil)

	err := svc.ReactToReel(ctx, reelID, userID, reactionType)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestReactToReel_ToggleOff(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	reactionType := models.ReactionLike

	existingReaction := &models.Reaction{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		TargetID: reelID,
		Type:     reactionType,
	}

	mockRepo.On("GetReaction", ctx, reelID, userID).Return(existingReaction, nil)
	mockRepo.On("RemoveReaction", ctx, existingReaction).Return(nil)

	err := svc.ReactToReel(ctx, reelID, userID, reactionType)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "AddReaction")
}

func TestReactToReel_ChangeReaction(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	oldReactionType := models.ReactionLike
	newReactionType := models.ReactionLove

	existingReaction := &models.Reaction{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		TargetID: reelID,
		Type:     oldReactionType,
	}

	mockRepo.On("GetReaction", ctx, reelID, userID).Return(existingReaction, nil)
	mockRepo.On("RemoveReaction", ctx, existingReaction).Return(nil)
	mockRepo.On("AddReaction", ctx, mock.MatchedBy(func(r *models.Reaction) bool {
		return r.Type == newReactionType
	})).Return(nil)

	err := svc.ReactToReel(ctx, reelID, userID, newReactionType)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ==================== AddComment Tests ====================

func TestAddComment_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	content := "Great reel!"
	author := models.PostAuthor{
		ID:       userID.Hex(),
		Username: "testuser",
	}

	mockRepo.On("AddComment", ctx, reelID, mock.AnythingOfType("models.Comment")).Return(nil)

	comment, err := svc.AddComment(ctx, reelID, userID, content, author, nil)

	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, content, comment.Content)
	assert.Equal(t, userID, comment.UserID)
	mockRepo.AssertExpectations(t)
}

func TestAddComment_WithMentions(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	mentionedUserID := primitive.NewObjectID()
	content := "Check this out!"
	author := models.PostAuthor{ID: userID.Hex()}
	explicitMentions := []primitive.ObjectID{mentionedUserID}

	mockRepo.On("AddComment", ctx, reelID, mock.MatchedBy(func(c models.Comment) bool {
		return len(c.Mentions) > 0
	})).Return(nil)

	comment, err := svc.AddComment(ctx, reelID, userID, content, author, explicitMentions)

	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Contains(t, comment.Mentions, mentionedUserID)
	mockRepo.AssertExpectations(t)
}

// ==================== GetComments Tests ====================

func TestGetComments_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	expectedComments := []models.Comment{
		{ID: primitive.NewObjectID(), Content: "Comment 1"},
		{ID: primitive.NewObjectID(), Content: "Comment 2"},
	}

	mockRepo.On("GetComments", ctx, reelID, int64(20), int64(0)).Return(expectedComments, nil)

	comments, err := svc.GetComments(ctx, reelID, 0, 0)

	assert.NoError(t, err)
	assert.Len(t, comments, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetComments_WithPagination(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	limit := int64(10)
	offset := int64(5)

	mockRepo.On("GetComments", ctx, reelID, limit, offset).Return([]models.Comment{}, nil)

	_, err := svc.GetComments(ctx, reelID, limit, offset)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ==================== AddReply Tests ====================

func TestAddReply_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	commentID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	content := "Nice comment!"
	author := models.PostAuthor{ID: userID.Hex()}

	mockRepo.On("AddReply", ctx, reelID, commentID, mock.AnythingOfType("models.Reply")).Return(nil)

	reply, err := svc.AddReply(ctx, reelID, commentID, userID, content, author)

	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, content, reply.Content)
	assert.Equal(t, commentID, reply.CommentID)
	mockRepo.AssertExpectations(t)
}

// ==================== ReactToComment Tests ====================

func TestReactToComment_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	commentID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	reactionType := models.ReactionLike

	mockRepo.On("ReactToComment", ctx, reelID, commentID, userID, reactionType).Return(nil)

	err := svc.ReactToComment(ctx, reelID, commentID, userID, reactionType)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ==================== GetUserReels Tests ====================

func TestGetUserReels_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	expectedReels := []models.Reel{
		{ID: primitive.NewObjectID(), UserID: userID, Caption: "Reel 1"},
		{ID: primitive.NewObjectID(), UserID: userID, Caption: "Reel 2"},
	}

	mockRepo.On("GetUserReels", ctx, userID).Return(expectedReels, nil)

	reels, err := svc.GetUserReels(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, reels, 2)
	mockRepo.AssertExpectations(t)
}

// ==================== ParseMentions Tests ====================

func TestParseMentions_NoUserClient(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	content := "Hello @testuser and @anotheruser!"

	// Without user client, should return empty (can't resolve usernames)
	mentions := svc.ParseMentions(ctx, content)

	assert.Empty(t, mentions)
}

// ==================== GetReelsFeed Tests ====================

func TestGetReelsFeed_Success(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	expectedReels := []models.Reel{
		{ID: primitive.NewObjectID(), Privacy: models.PrivacySettingPublic},
	}

	mockRepo.On("GetReelsFeed", ctx, userID, []primitive.ObjectID{}, int64(10), int64(0)).Return(expectedReels, nil)

	reels, err := svc.GetReelsFeed(ctx, userID, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, reels, 1)
	mockRepo.AssertExpectations(t)
}

// ==================== Edge Cases ====================

func TestCreateReel_RepoError(t *testing.T) {
	mockRepo := new(MockReelRepository)
	mockBroadcaster := new(MockBroadcaster)
	svc := newTestReelService(mockRepo, mockBroadcaster)

	ctx := context.Background()
	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}
	req := CreateReelRequest{
		VideoURL: "https://example.com/video.mp4",
	}

	mockRepo.On("CreateReel", ctx, mock.Anything).Return(nil, assert.AnError)

	reel, err := svc.CreateReel(ctx, userID, author, req)

	assert.Error(t, err)
	assert.Nil(t, reel)
	mockBroadcaster.AssertNotCalled(t, "PublishReelCreated")
}

func TestGetComments_NegativeOffset(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()

	// Negative offset should be normalized to 0
	mockRepo.On("GetComments", ctx, reelID, int64(20), int64(0)).Return([]models.Comment{}, nil)

	_, err := svc.GetComments(ctx, reelID, 0, -5) // Negative offset

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ==================== Timestamp Tests ====================

func TestAddComment_SetsCreatedAt(t *testing.T) {
	mockRepo := new(MockReelRepository)
	svc := newTestReelService(mockRepo, nil)

	ctx := context.Background()
	reelID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}

	before := time.Now()

	mockRepo.On("AddComment", ctx, reelID, mock.MatchedBy(func(c models.Comment) bool {
		return !c.CreatedAt.IsZero() && c.CreatedAt.After(before.Add(-time.Second))
	})).Return(nil)

	comment, err := svc.AddComment(ctx, reelID, userID, "Test", author, nil)

	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.False(t, comment.CreatedAt.IsZero())
}
