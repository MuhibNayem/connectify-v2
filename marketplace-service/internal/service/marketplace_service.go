package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/validation"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MarketplaceRepository defines the data access layer interface
type MarketplaceRepository interface {
	GetCategories(ctx context.Context) ([]models.Category, error)
	CreateProduct(ctx context.Context, product *models.Product) (*models.Product, error)
	GetProductByID(ctx context.Context, id primitive.ObjectID) (*models.Product, error)
	ListProducts(ctx context.Context, filter models.ProductFilter) ([]models.ProductResponse, int64, error)
	GetMarketplaceConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationSummary, error)
	UpdateProduct(ctx context.Context, id primitive.ObjectID, update bson.M) (*models.Product, error)
	DeleteProduct(ctx context.Context, id primitive.ObjectID) error
	IncrementViews(ctx context.Context, id primitive.ObjectID) error
}

type MarketplaceService struct {
	repo    MarketplaceRepository
	metrics *metrics.BusinessMetrics
	logger  *slog.Logger
	cb      *resilience.CircuitBreaker // Optional: If we have external service calls
}

func NewMarketplaceService(
	repo MarketplaceRepository,
	metrics *metrics.BusinessMetrics,
	logger *slog.Logger,
	cb *resilience.CircuitBreaker,
) *MarketplaceService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MarketplaceService{
		repo:    repo,
		metrics: metrics,
		logger:  logger,
		cb:      cb,
	}
}

func (s *MarketplaceService) GetCategories(ctx context.Context) ([]models.Category, error) {
	return s.repo.GetCategories(ctx)
}

func (s *MarketplaceService) CreateProduct(ctx context.Context, userID primitive.ObjectID, req models.CreateProductRequest) (*models.Product, error) {
	// 1. Validation
	if err := validation.ValidateCreateProductRequest(&req); err != nil {
		s.logger.Warn("Invalid create product request", "error", err, "user_id", userID)
		return nil, err
	}

	catID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, errors.New("invalid category ID")
	}

	product := &models.Product{
		SellerID:    userID,
		CategoryID:  catID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		Images:      req.Images,
		Location:    models.ProductLocation{City: req.Location}, // Convert string to structured location
		Status:      models.ProductStatusAvailable,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdProduct, err := s.repo.CreateProduct(ctx, product)
	if err != nil {
		s.logger.Error("Failed to create product", "error", err, "user_id", userID)
		return nil, err
	}

	// Metrics
	s.metrics.IncrementProductsCreated()
	s.logger.Info("Product created", "product_id", createdProduct.ID, "user_id", userID)

	return createdProduct, nil
}

func (s *MarketplaceService) GetProductByID(ctx context.Context, id primitive.ObjectID, viewerID primitive.ObjectID) (*models.ProductResponse, error) {
	// Async view increment with error logging safety
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.IncrementViews(bgCtx, id); err != nil {
			s.logger.Warn("Failed to increment views", "error", err, "product_id", id)
		} else {
			s.metrics.IncrementProductViews()
		}
	}()

	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	categories, err := s.repo.GetCategories(ctx)
	var category models.Category
	if err == nil {
		for _, cat := range categories {
			if cat.ID == product.CategoryID {
				category = cat
				break
			}
		}
	} else {
		s.logger.Warn("Failed to fetch categories during product get", "error", err)
	}

	isSaved := false
	for _, savedID := range product.SavedBy {
		if savedID == viewerID {
			isSaved = true
			break
		}
	}

	return &models.ProductResponse{
		ID:          product.ID,
		Title:       product.Title,
		Description: product.Description,
		Price:       product.Price,
		Currency:    product.Currency,
		Images:      product.Images,
		Location:    product.Location,
		Status:      product.Status,
		Tags:        product.Tags,
		Views:       product.Views,
		CreatedAt:   product.CreatedAt,
		Category:    category,
		IsSaved:     isSaved,
	}, nil
}

type MarketplaceListResponse struct {
	Products []models.ProductResponse `json:"products"`
	Total    int64                    `json:"total"`
	Page     int64                    `json:"page"`
	Limit    int64                    `json:"limit"`
}

func (s *MarketplaceService) SearchProducts(ctx context.Context, filter models.ProductFilter) (*MarketplaceListResponse, error) {
	products, total, err := s.repo.ListProducts(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to search products", "error", err)
		return nil, err
	}

	return &MarketplaceListResponse{
		Products: products,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
	}, nil
}

func (s *MarketplaceService) GetMarketplaceConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationSummary, error) {
	return s.repo.GetMarketplaceConversations(ctx, userID)
}

func (s *MarketplaceService) MarkProductSold(ctx context.Context, productID, userID primitive.ObjectID) error {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	if product.SellerID != userID {
		return errors.New("unauthorized")
	}

	_, err = s.repo.UpdateProduct(ctx, productID, bson.M{
		"status":     models.ProductStatusSold,
		"updated_at": time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to mark product sold", "error", err, "product_id", productID)
		return err
	}

	s.metrics.IncrementProductsSold()
	s.logger.Info("Product marked as sold", "product_id", productID, "user_id", userID)
	return nil
}

func (s *MarketplaceService) DeleteProduct(ctx context.Context, productID, userID primitive.ObjectID) error {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}
	if product.SellerID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.DeleteProduct(ctx, productID); err != nil {
		s.logger.Error("Failed to delete product", "error", err, "product_id", productID)
		return err
	}

	s.metrics.IncrementProductsDeleted()
	s.logger.Info("Product deleted", "product_id", productID, "user_id", userID)
	return nil
}

func (s *MarketplaceService) ToggleSaveProduct(ctx context.Context, productID, userID primitive.ObjectID) (bool, error) {
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return false, err
	}

	isSaved := false
	for _, id := range product.SavedBy {
		if id == userID {
			isSaved = true
			break
		}
	}

	var update bson.M
	if isSaved {
		update = bson.M{"$pull": bson.M{"saved_by": userID}}
	} else {
		update = bson.M{"$addToSet": bson.M{"saved_by": userID}}
	}
	update["updated_at"] = time.Now()

	_, err = s.repo.UpdateProduct(ctx, productID, update)
	if err != nil {
		s.logger.Error("Failed to toggle save product", "error", err, "product_id", productID)
		return false, err
	}

	return !isSaved, nil
}
