package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/producer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockStoryRepository struct {
	mock.Mock
}

func (m *MockStoryRepository) CreateStory(ctx context.Context, story *models.Story) (*models.Story, error) {
	args := m.Called(ctx, story)
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *MockStoryRepository) GetStoryByID(ctx context.Context, id primitive.ObjectID) (*models.Story, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *MockStoryRepository) DeleteStory(ctx context.Context, id primitive.ObjectID, userID primitive.ObjectID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockStoryRepository) GetActiveStoryAuthors(ctx context.Context, viewerID primitive.ObjectID, userIDs []primitive.ObjectID, limit, offset int) ([]primitive.ObjectID, error) {
	args := m.Called(ctx, viewerID, userIDs, limit, offset)
	return args.Get(0).([]primitive.ObjectID), args.Error(1)
}

func (m *MockStoryRepository) GetStoriesForUsers(ctx context.Context, viewerID primitive.ObjectID, authorIDs []primitive.ObjectID) ([]models.Story, error) {
	args := m.Called(ctx, viewerID, authorIDs)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *MockStoryRepository) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *MockStoryRepository) AddViewer(ctx context.Context, storyID primitive.ObjectID, viewerID primitive.ObjectID) error {
	args := m.Called(ctx, storyID, viewerID)
	return args.Error(0)
}

func (m *MockStoryRepository) AddReaction(ctx context.Context, storyID primitive.ObjectID, reaction models.StoryReaction) error {
	args := m.Called(ctx, storyID, reaction)
	return args.Error(0)
}

func (m *MockStoryRepository) GetStoryViewersWithReactions(ctx context.Context, storyID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	args := m.Called(ctx, storyID)
	return args.Get(0).([]models.StoryViewerResponse), args.Error(1)
}

type MockBroadcaster struct {
	mock.Mock
}

func (m *MockBroadcaster) PublishStoryCreated(ctx context.Context, event producer.StoryCreatedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) PublishStoryDeleted(ctx context.Context, event producer.StoryDeletedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) PublishStoryViewed(ctx context.Context, event producer.StoryViewedEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) PublishStoryReaction(ctx context.Context, event producer.StoryReactionEvent) {
	m.Called(ctx, event)
}

func (m *MockBroadcaster) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestStoryService_CreateStory(t *testing.T) {
	mockRepo := new(MockStoryRepository)
	mockBroadcaster := new(MockBroadcaster)
	businessMetrics := metrics.NewBusinessMetrics()

	service := NewStoryService(
		mockRepo,
		mockBroadcaster,
		nil,
		nil,
		businessMetrics,
		slog.Default(),
		nil,
	)

	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}

	req := CreateStoryRequest{
		MediaURL:  "https://example.com/story.jpg",
		MediaType: "image",
		Privacy:   models.PrivacySettingPublic,
	}

	expectedStory := &models.Story{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		Author:    author,
		MediaURL:  req.MediaURL,
		MediaType: req.MediaType,
		Privacy:   req.Privacy,
	}

	mockRepo.On("CreateStory", mock.Anything, mock.AnythingOfType("*models.Story")).Return(expectedStory, nil)
	mockBroadcaster.On("PublishStoryCreated", mock.Anything, mock.Anything).Return()

	story, err := service.CreateStory(context.Background(), userID, author, req)

	assert.NoError(t, err)
	assert.NotNil(t, story)
	assert.Equal(t, expectedStory.ID, story.ID)
	assert.Equal(t, userID, story.UserID)

	mockRepo.AssertExpectations(t)
	mockBroadcaster.AssertExpectations(t)
}

func TestStoryService_GetStoryViewers_Unauthorized(t *testing.T) {
	mockRepo := new(MockStoryRepository)
	service := NewStoryService(mockRepo, nil, nil, nil, nil, slog.Default(), nil)

	storyID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	differentUserID := primitive.NewObjectID()

	story := &models.Story{
		ID:     storyID,
		UserID: differentUserID, // Different user owns the story
	}

	mockRepo.On("GetStoryByID", mock.Anything, storyID).Return(story, nil)

	viewers, err := service.GetStoryViewers(context.Background(), storyID, userID)

	assert.Error(t, err)
	assert.Nil(t, viewers)
	assert.Contains(t, err.Error(), "unauthorized")

	mockRepo.AssertExpectations(t)
}

func TestStoryService_CreateStory_ValidationError(t *testing.T) {
	service := NewStoryService(nil, nil, nil, nil, nil, slog.Default(), nil)

	userID := primitive.NewObjectID()
	author := models.PostAuthor{ID: userID.Hex()}

	req := CreateStoryRequest{
		MediaURL:  "", // Invalid - empty URL
		MediaType: "image",
	}

	story, err := service.CreateStory(context.Background(), userID, author, req)

	assert.Error(t, err)
	assert.Nil(t, story)
	assert.Contains(t, err.Error(), "media URL is required")
}
