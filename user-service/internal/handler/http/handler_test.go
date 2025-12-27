package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"user-service/config"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*models.AuthResponse, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func TestAuthHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAuthService := new(MockAuthService)
	cfg := &config.Config{
		RefreshCookieName: "refresh_token",
		RefreshTokenTTL:   time.Hour * 24 * 7,
	}
	handler := NewAuthHandler(mockAuthService, cfg)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	authResponse := &models.AuthResponse{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		User: models.SafeUserResponse{
			ID:       primitive.NewObjectID(),
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	mockAuthService.On("Register", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Username == "testuser" && u.Email == "test@example.com"
	})).Return(authResponse, nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/register", handler.Register)

	userJSON, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/register", bytes.NewReader(userJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "access_token", response.AccessToken)
	assert.Equal(t, "testuser", response.User.Username)

	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAuthService := new(MockAuthService)
	handler := NewAuthHandler(mockAuthService, &config.Config{})

	// Invalid JSON
	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/register", handler.Register)

	req := httptest.NewRequest("POST", "/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid")
}

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAuthService := new(MockAuthService)
	cfg := &config.Config{
		RefreshCookieName: "refresh_token",
		RefreshTokenTTL:   time.Hour * 24 * 7,
	}
	handler := NewAuthHandler(mockAuthService, cfg)

	authResponse := &models.AuthResponse{
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		User: models.SafeUserResponse{
			ID:       primitive.NewObjectID(),
			Username: "testuser",
			Email:    "test@example.com",
		},
	}

	mockAuthService.On("Login", mock.Anything, "test@example.com", "password123").Return(authResponse, nil)

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/login", handler.Login)

	creds := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	credsJSON, _ := json.Marshal(creds)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(credsJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "access_token", response.AccessToken)

	mockAuthService.AssertExpectations(t)
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAuthService := new(MockAuthService)
	handler := NewAuthHandler(mockAuthService, &config.Config{})

	mockAuthService.On("Login", mock.Anything, "test@example.com", "wrongpassword").Return(nil, errors.New("invalid credentials"))

	w := httptest.NewRecorder()
	router := gin.New()
	router.POST("/login", handler.Login)

	creds := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}
	credsJSON, _ := json.Marshal(creds)
	req := httptest.NewRequest("POST", "/login", bytes.NewReader(credsJSON))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid credentials")

	mockAuthService.AssertExpectations(t)
}
