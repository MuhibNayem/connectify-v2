package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockMarketplaceService struct {
	mock.Mock
}

func (m *MockMarketplaceService) GetCategories(ctx context.Context) ([]models.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockMarketplaceService) CreateProduct(ctx context.Context, userID primitive.ObjectID, req models.CreateProductRequest) (*models.Product, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockMarketplaceService) GetProductByID(ctx context.Context, id primitive.ObjectID, viewerID primitive.ObjectID) (*models.ProductResponse, error) {
	args := m.Called(ctx, id, viewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductResponse), args.Error(1)
}

func (m *MockMarketplaceService) SearchProducts(ctx context.Context, filter models.ProductFilter) (*service.MarketplaceListResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.MarketplaceListResponse), args.Error(1)
}

func (m *MockMarketplaceService) MarkProductSold(ctx context.Context, productID, userID primitive.ObjectID) error {
	args := m.Called(ctx, productID, userID)
	return args.Error(0)
}

func (m *MockMarketplaceService) DeleteProduct(ctx context.Context, productID, userID primitive.ObjectID) error {
	args := m.Called(ctx, productID, userID)
	return args.Error(0)
}

func (m *MockMarketplaceService) ToggleSaveProduct(ctx context.Context, productID, userID primitive.ObjectID) (bool, error) {
	args := m.Called(ctx, productID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMarketplaceService) GetMarketplaceConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationSummary, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.ConversationSummary), args.Error(1)
}

func TestMarketplaceController_CreateProduct_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := new(MockMarketplaceService)
	controller := NewMarketplaceController(mockService)
	
	userID := primitive.NewObjectID()
	productID := primitive.NewObjectID()
	
	expectedProduct := &models.Product{
		ID:          productID,
		SellerID:    userID,
		Title:       "iPhone 15 Pro",
		Description: "Brand new iPhone",
		Price:       999.99,
		Currency:    "USD",
		Images:      []string{"https://example.com/image1.jpg"},
		Location:    models.ProductLocation{City: "New York"},
		Status:      models.ProductStatusAvailable,
	}
	
	mockService.On("CreateProduct", mock.Anything, userID, mock.AnythingOfType("models.CreateProductRequest")).Return(expectedProduct, nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})
	
	router.POST("/products", controller.CreateProduct)
	
	reqBody := models.CreateProductRequest{
		Title:       "iPhone 15 Pro",
		Description: "Brand new iPhone",
		Price:       999.99,
		Currency:    "USD",
		Images:      []string{"https://example.com/image1.jpg"},
		Location:    "New York",
		CategoryID:  primitive.NewObjectID().Hex(),
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Product created successfully", response.Message)
	
	mockService.AssertExpectations(t)
}

func TestMarketplaceController_CreateProduct_TooManyImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := new(MockMarketplaceService)
	controller := NewMarketplaceController(mockService)
	
	userID := primitive.NewObjectID()
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})
	
	router.POST("/products", controller.CreateProduct)
	
	reqBody := models.CreateProductRequest{
		Title:       "Test Product",
		Description: "Test description",
		Price:       100.00,
		Currency:    "USD",
		Images:      []string{"1", "2", "3", "4", "5", "6"}, // 6 images (> 5 limit)
		Location:    "Test Location",
		CategoryID:  primitive.NewObjectID().Hex(),
	}
	reqJSON, _ := json.Marshal(reqBody)
	
	req := httptest.NewRequest("POST", "/products", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Too many images", response.Error)
	assert.Equal(t, ErrCodeValidation, response.Code)
	assert.Contains(t, response.Details["images"], "Maximum 5 images")
}

func TestMarketplaceController_GetProduct_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := new(MockMarketplaceService)
	controller := NewMarketplaceController(mockService)
	
	productID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	expectedProduct := &models.ProductResponse{
		ID:          productID,
		Title:       "iPhone 15 Pro",
		Description: "Brand new iPhone",
		Price:       999.99,
		Currency:    "USD",
		Images:      []string{"https://example.com/image1.jpg"},
		IsSaved:     false,
	}
	
	mockService.On("GetProductByID", mock.Anything, productID, userID).Return(expectedProduct, nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware (optional for GET)
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})
	
	router.GET("/products/:id", controller.GetProduct)
	
	req := httptest.NewRequest("GET", "/products/"+productID.Hex(), nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ProductResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, productID, response.ID)
	assert.Equal(t, "iPhone 15 Pro", response.Title)
	
	mockService.AssertExpectations(t)
}

func TestMarketplaceController_MarkProductSold_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := new(MockMarketplaceService)
	controller := NewMarketplaceController(mockService)
	
	productID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	mockService.On("MarkProductSold", mock.Anything, productID, userID).Return(errors.New("unauthorized"))
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})
	
	router.PUT("/products/:id/sold", controller.MarkProductSold)
	
	req := httptest.NewRequest("PUT", "/products/"+productID.Hex()+"/sold", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "You can only mark your own products as sold", response.Error)
	assert.Equal(t, ErrCodeInsufficientPerms, response.Code)
	
	mockService.AssertExpectations(t)
}

func TestMarketplaceController_ToggleSaveProduct_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockService := new(MockMarketplaceService)
	controller := NewMarketplaceController(mockService)
	
	productID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	
	mockService.On("ToggleSaveProduct", mock.Anything, productID, userID).Return(true, nil)
	
	w := httptest.NewRecorder()
	router := gin.New()
	
	// Mock authentication middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID.Hex())
		c.Next()
	})
	
	router.PUT("/products/:id/save", controller.ToggleSaveProduct)
	
	req := httptest.NewRequest("PUT", "/products/"+productID.Hex()+"/save", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Product saved successfully", response.Message)
	
	// Check the data field
	data := response.Data.(map[string]interface{})
	assert.Equal(t, true, data["saved"])
	
	mockService.AssertExpectations(t)
}