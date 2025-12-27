package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockStoryService struct {
	mock.Mock
}

func (m *MockStoryService) CreateStory(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor, req service.CreateStoryRequest) (*models.Story, error) {
	args := m.Called(ctx, userID, author, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *MockStoryService) GetStory(ctx context.Context, storyID, viewerID primitive.ObjectID) (*models.Story, error) {
	args := m.Called(ctx, storyID, viewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Story), args.Error(1)
}

func (m *MockStoryService) DeleteStory(ctx context.Context, storyID, userID primitive.ObjectID) error {
	args := m.Called(ctx, storyID, userID)
	return args.Error(0)
}

func (m *MockStoryService) RecordView(ctx context.Context, storyID, viewerID primitive.ObjectID) error {
	args := m.Called(ctx, storyID, viewerID)
	return args.Error(0)
}

func (m *MockStoryService) ReactToStory(ctx context.Context, storyID, userID primitive.ObjectID, reactionType string) error {
	args := m.Called(ctx, storyID, userID, reactionType)
	return args.Error(0)
}

func (m *MockStoryService) GetStoriesFeed(ctx context.Context, viewerID primitive.ObjectID, friendIDs []primitive.ObjectID, limit, offset int) ([]models.Story, error) {
	args := m.Called(ctx, viewerID, friendIDs, limit, offset)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *MockStoryService) GetUserStories(ctx context.Context, userID primitive.ObjectID) ([]models.Story, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Story), args.Error(1)
}

func (m *MockStoryService) GetStoryViewers(ctx context.Context, storyID, userID primitive.ObjectID) ([]models.StoryViewerResponse, error) {
	args := m.Called(ctx, storyID, userID)
	return args.Get(0).([]models.StoryViewerResponse), args.Error(1)
}

func TestStoryHandler_CreateStory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockStoryService)
	businessMetrics := metrics.NewBusinessMetrics()
	handler := NewStoryHandler(mockService, nil, businessMetrics)

	userID := primitive.NewObjectID()
	storyID := primitive.NewObjectID()

	expectedStory := &models.Story{
		ID:        storyID,
		UserID:    userID,
		MediaURL:  "https://example.com/story.jpg",
		MediaType: "image",
		Privacy:   models.PrivacySettingFriends,
	}

	mockService.On("CreateStory", mock.Anything, userID, mock.AnythingOfType("models.PostAuthor"), mock.AnythingOfType("service.CreateStoryRequest")).Return(expectedStory, nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.POST("/stories", handler.CreateStory)

	reqBody := createStoryRequest{
		MediaURL:  "https://example.com/story.jpg",
		MediaType: "image",
		Privacy:   "friends",
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/stories", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Story created successfully", response.Message)

	mockService.AssertExpectations(t)
}

func TestStoryHandler_CreateStory_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockStoryService)
	handler := NewStoryHandler(mockService, nil, nil)

	userID := primitive.NewObjectID()

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.POST("/stories", handler.CreateStory)

	// Missing required fields
	reqBody := createStoryRequest{
		MediaURL:  "", // Missing
		MediaType: "", // Missing
	}
	reqJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/stories", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing required fields", response.Error)
	assert.Equal(t, ErrCodeValidation, response.Code)
	assert.NotEmpty(t, response.Details)
}

func TestStoryHandler_GetStory_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockStoryService)
	handler := NewStoryHandler(mockService, nil, nil)

	userID := primitive.NewObjectID()
	storyID := primitive.NewObjectID()

	expectedStory := &models.Story{
		ID:        storyID,
		UserID:    userID,
		MediaURL:  "https://example.com/story.jpg",
		MediaType: "image",
		Privacy:   models.PrivacySettingPublic,
	}

	mockService.On("GetStory", mock.Anything, storyID, userID).Return(expectedStory, nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.GET("/stories/:id", handler.GetStory)

	req := httptest.NewRequest("GET", "/stories/"+storyID.Hex(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.Story
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, storyID, response.ID)

	mockService.AssertExpectations(t)
}

func TestStoryHandler_GetStory_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockStoryService)
	handler := NewStoryHandler(mockService, nil, nil)

	storyID := primitive.NewObjectID()

	w := httptest.NewRecorder()
	router := gin.New()

	// No authentication middleware - user_id not set
	router.GET("/stories/:id", handler.GetStory)

	req := httptest.NewRequest("GET", "/stories/"+storyID.Hex(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Authentication required", response.Error)
	assert.Equal(t, ErrCodeUnauthorized, response.Code)
}

func TestStoryHandler_GetStory_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := new(MockStoryService)
	handler := NewStoryHandler(mockService, nil, nil)

	userID := primitive.NewObjectID()
	storyID := primitive.NewObjectID()

	mockService.On("GetStory", mock.Anything, storyID, userID).Return(nil, errors.New("story not found"))

	w := httptest.NewRecorder()
	router := gin.New()

	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.GET("/stories/:id", handler.GetStory)

	req := httptest.NewRequest("GET", "/stories/"+storyID.Hex(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Story not found or access denied", response.Error)
	assert.Equal(t, ErrCodeStoryNotFound, response.Code)

	mockService.AssertExpectations(t)
}
