package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) CreateEvent(ctx context.Context, userID primitive.ObjectID, req models.CreateEventRequest) (*models.Event, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) GetEvent(ctx context.Context, eventID, userID primitive.ObjectID) (*models.EventResponse, error) {
	args := m.Called(ctx, eventID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EventResponse), args.Error(1)
}

func (m *MockEventService) UpdateEvent(ctx context.Context, eventID, userID primitive.ObjectID, req models.UpdateEventRequest) (*models.Event, error) {
	args := m.Called(ctx, eventID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

func (m *MockEventService) DeleteEvent(ctx context.Context, eventID, userID primitive.ObjectID) error {
	args := m.Called(ctx, eventID, userID)
	return args.Error(0)
}

func (m *MockEventService) ListEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64, query, category, period string) ([]models.EventResponse, int64, error) {
	args := m.Called(ctx, userID, limit, page, query, category, period)
	return args.Get(0).([]models.EventResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockEventService) GetUserEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.EventResponse, error) {
	args := m.Called(ctx, userID, limit, page)
	return args.Get(0).([]models.EventResponse), args.Error(1)
}

func (m *MockEventService) RSVP(ctx context.Context, eventID, userID primitive.ObjectID, status string) error {
	args := m.Called(ctx, eventID, userID, status)
	return args.Error(0)
}

type MockRecommendationService struct {
	mock.Mock
}

func (m *MockRecommendationService) GetRecommendations(ctx context.Context, userID primitive.ObjectID, limit int) ([]models.EventResponse, error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]models.EventResponse), args.Error(1)
}

func TestEventController_CreateEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	userID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	
	expectedEvent := &models.Event{
		ID:          eventID,
		OrganizerID: userID,
		Title:       "Test Event",
		Description: "Test Description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		Location:    "Test Location",
		Privacy:     models.EventPrivacyPublic,
	}
	
	mockEventService.On("CreateEvent", mock.Anything, userID, mock.AnythingOfType("models.CreateEventRequest")).Return(expectedEvent, nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	
	router.POST("/events", controller.CreateEvent)
	
	reqBody := models.CreateEventRequest{
		Title:       "Test Event",
		Description: "Test Description",
		StartDate:   time.Now().Add(24 * time.Hour),
		EndDate:     time.Now().Add(26 * time.Hour),
		Location:    "Test Location",
		Privacy:     models.EventPrivacyPublic,
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/events", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Event created successfully", response["message"])
	
	mockEventService.AssertExpectations(t)
}

func TestEventController_CreateEvent_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// No authentication middleware - userID not set
	router.POST("/events", controller.CreateEvent)
	
	reqBody := models.CreateEventRequest{
		Title:       "Test Event",
		Description: "Test Description",
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/events", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Authentication required")
}

func TestEventController_GetEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	userID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	
	expectedEvent := &models.EventResponse{
		ID:          eventID,
		Title:       "Test Event",
		Description: "Test Description",
		Location:    "Test Location",
		Privacy:     models.EventPrivacyPublic,
		IsAttending: false,
	}
	
	mockEventService.On("GetEvent", mock.Anything, eventID, userID).Return(expectedEvent, nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware (optional for GET)
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	
	router.GET("/events/:id", controller.GetEvent)
	
	req := httptest.NewRequest("GET", "/events/"+eventID.Hex(), nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Event retrieved successfully", response["message"])
	
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Test Event", data["title"])
	
	mockEventService.AssertExpectations(t)
}

func TestEventController_DeleteEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	userID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	
	mockEventService.On("DeleteEvent", mock.Anything, eventID, userID).Return(nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	
	router.DELETE("/events/:id", controller.DeleteEvent)
	
	req := httptest.NewRequest("DELETE", "/events/"+eventID.Hex(), nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Event deleted successfully", response["message"])
	
	mockEventService.AssertExpectations(t)
}

func TestEventController_DeleteEvent_Unauthorized_NotOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	userID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	
	mockEventService.On("DeleteEvent", mock.Anything, eventID, userID).Return(errors.New("unauthorized: not event organizer"))
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	
	router.DELETE("/events/:id", controller.DeleteEvent)
	
	req := httptest.NewRequest("DELETE", "/events/"+eventID.Hex(), nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "unauthorized")
	
	mockEventService.AssertExpectations(t)
}

func TestEventController_RSVP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockEventService := new(MockEventService)
	mockRecoService := new(MockRecommendationService)
	controller := NewEventController(mockEventService, mockRecoService)
	
	userID := primitive.NewObjectID()
	eventID := primitive.NewObjectID()
	
	mockEventService.On("RSVP", mock.Anything, eventID, userID, "attending").Return(nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	
	router.POST("/events/:id/rsvp", controller.RSVP)
	
	reqBody := map[string]string{
		"status": "attending",
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/events/"+eventID.Hex()+"/rsvp", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "RSVP updated successfully", response["message"])
	
	mockEventService.AssertExpectations(t)
}