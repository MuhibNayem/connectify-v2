package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateProfileFields(ctx context.Context, userID primitive.ObjectID, fullName, bio, avatar, coverPhoto, location, website string) (*models.User, error) {
	args := m.Called(ctx, userID, fullName, bio, avatar, coverPhoto, location, website)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) UpdateEmail(ctx context.Context, userID primitive.ObjectID, email string) error {
	args := m.Called(ctx, userID, email)
	return args.Error(0)
}

func (m *MockUserService) UpdatePassword(ctx context.Context, userID primitive.ObjectID, currentPassword, newPassword string) error {
	args := m.Called(ctx, userID, currentPassword, newPassword)
	return args.Error(0)
}

func (m *MockUserService) UpdatePrivacySettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdatePrivacySettingsRequest) error {
	args := m.Called(ctx, userID, settings)
	return args.Error(0)
}

func (m *MockUserService) UpdateNotificationSettings(ctx context.Context, userID primitive.ObjectID, settings *models.UpdateNotificationSettingsRequest) error {
	args := m.Called(ctx, userID, settings)
	return args.Error(0)
}

func (m *MockUserService) ToggleTwoFactor(ctx context.Context, userID primitive.ObjectID, enable bool) error {
	args := m.Called(ctx, userID, enable)
	return args.Error(0)
}

func (m *MockUserService) DeactivateAccount(ctx context.Context, userID primitive.ObjectID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetUserStatus(ctx context.Context, userIDStr string) (string, int64, error) {
	args := m.Called(ctx, userIDStr)
	return args.String(0), args.Get(1).(int64), args.Error(2)
}

func TestUserHandler_GetProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(MockUserService)
	handler := NewUserHandler(mockUserService)

	userID := primitive.NewObjectID()
	user := &models.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Test User",
		Password: "hashedpassword", // This should be cleared in response
	}

	mockUserService.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Simulate authentication middleware setting user ID
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.GET("/profile", handler.GetProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.User
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", response.Username)
	assert.Equal(t, "", response.Password) // Password should be cleared

	mockUserService.AssertExpectations(t)
}

func TestUserHandler_GetProfile_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(MockUserService)
	handler := NewUserHandler(mockUserService)

	w := httptest.NewRecorder()
	router := gin.New()

	// No user ID set in context (simulating missing auth)
	router.GET("/profile", handler.GetProfile)

	req := httptest.NewRequest("GET", "/profile", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Authentication required")
}

func TestUserHandler_GetUserByID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(MockUserService)
	handler := NewUserHandler(mockUserService)

	userID := primitive.NewObjectID()
	user := &models.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Test User",
		Bio:      "Test bio",
		Avatar:   "avatar.jpg",
		Password: "hashedpassword", // Should not be returned
	}

	mockUserService.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/users/:id", handler.GetUserByID)

	req := httptest.NewRequest("GET", "/users/"+userID.Hex(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", response["username"])
	assert.Equal(t, "Test User", response["full_name"])
	assert.Equal(t, "Test bio", response["bio"])

	// Password should not be in public profile
	_, exists := response["password"]
	assert.False(t, exists)

	mockUserService.AssertExpectations(t)
}

func TestUserHandler_GetUserByID_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(MockUserService)
	handler := NewUserHandler(mockUserService)

	w := httptest.NewRecorder()
	router := gin.New()
	router.GET("/users/:id", handler.GetUserByID)

	req := httptest.NewRequest("GET", "/users/invalid-id", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid user ID format")
}

func TestUserHandler_UpdateProfile_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserService := new(MockUserService)
	handler := NewUserHandler(mockUserService)

	userID := primitive.NewObjectID()

	mockUserService.On("UpdateProfileFields",
		mock.Anything,
		userID,
		"Updated Name",
		"Updated bio",
		"", "", "", "" /* other fields empty */).Return(&models.User{}, nil)

	w := httptest.NewRecorder()
	router := gin.New()

	// Simulate authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID.Hex())
		c.Next()
	})

	router.PUT("/profile", handler.UpdateProfile)

	updateData := map[string]string{
		"full_name": "Updated Name",
		"bio":       "Updated bio",
	}
	updateJSON, _ := json.Marshal(updateData)
	req := httptest.NewRequest("PUT", "/profile", bytes.NewReader(updateJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "profile updated successfully", response["message"].(string))

	mockUserService.AssertExpectations(t)
}
