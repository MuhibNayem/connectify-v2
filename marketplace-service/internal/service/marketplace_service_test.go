package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockMarketplaceRepository struct {
	mock.Mock
}

func (m *MockMarketplaceRepository) GetCategories(ctx context.Context) ([]models.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockMarketplaceRepository) CreateProduct(ctx context.Context, product *models.Product) (*models.Product, error) {
	args := m.Called(ctx, product)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockMarketplaceRepository) GetProductByID(ctx context.Context, id primitive.ObjectID) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockMarketplaceRepository) ListProducts(ctx context.Context, filter models.ProductFilter) ([]models.ProductResponse, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]models.ProductResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockMarketplaceRepository) GetMarketplaceConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationSummary, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.ConversationSummary), args.Error(1)
}

func (m *MockMarketplaceRepository) UpdateProduct(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.Product, error) {
	args := m.Called(ctx, id, update)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockMarketplaceRepository) DeleteProduct(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMarketplaceRepository) IncrementViews(ctx context.Context, id primitive.ObjectID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMarketplaceService_CreateProduct(t *testing.T) {
	mockRepo := new(MockMarketplaceRepository)
	businessMetrics := metrics.NewBusinessMetrics()
	service := NewMarketplaceService(mockRepo, businessMetrics, slog.Default(), nil, nil)

	userID := primitive.NewObjectID()
	categoryID := primitive.NewObjectID()

	req := models.CreateProductRequest{
		Title:       "Test Product",
		Description: "A test product",
		Price:       100.0,
		Currency:    "USD",
		CategoryID:  categoryID.Hex(),
		Images:      []string{"http://example.com/image.jpg"},
		Location:    "New York",
	}

	expectedProduct := &models.Product{
		ID:          primitive.NewObjectID(),
		SellerID:    userID,
		CategoryID:  categoryID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		Images:      req.Images,
		Location:    models.ProductLocation{City: req.Location},
		Status:      models.ProductStatusAvailable,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// We use mock.MatchedBy to match pointer content loosely or just verify the call arguments logic
	mockRepo.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
		return p.Title == req.Title && p.SellerID == userID
	})).Return(expectedProduct, nil)

	createdProduct, err := service.CreateProduct(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, createdProduct)
	assert.Equal(t, req.Title, createdProduct.Title)

	mockRepo.AssertExpectations(t)
}

func TestMarketplaceService_CreateProduct_ValidationFail(t *testing.T) {
	mockRepo := new(MockMarketplaceRepository)
	service := NewMarketplaceService(mockRepo, nil, slog.Default(), nil, nil)

	userID := primitive.NewObjectID()

	// Empty Title
	req := models.CreateProductRequest{
		Title:       "",
		Description: "A test product",
		Price:       100.0,
	}

	product, err := service.CreateProduct(context.Background(), userID, req)

	assert.Error(t, err)
	assert.Nil(t, product)
	assert.Contains(t, err.Error(), "title is required")

	mockRepo.AssertNotCalled(t, "CreateProduct")
}

func TestMarketplaceService_MarkProductSold_Unauthorized(t *testing.T) {
	mockRepo := new(MockMarketplaceRepository)
	service := NewMarketplaceService(mockRepo, nil, slog.Default(), nil, nil)

	productID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	existingProduct := &models.Product{
		ID:       productID,
		SellerID: otherUserID, // Different owner
		Status:   models.ProductStatusAvailable,
	}

	mockRepo.On("GetProductByID", mock.Anything, productID).Return(existingProduct, nil)

	err := service.MarkProductSold(context.Background(), productID, userID)

	assert.Error(t, err)
	assert.Equal(t, "unauthorized", err.Error())

	mockRepo.AssertExpectations(t)
}
