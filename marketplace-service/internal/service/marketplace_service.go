package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/validation"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryCache struct {
	sync.RWMutex
	categories []models.Category
	lastFetch  time.Time
	ttl        time.Duration
}

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
	repo          MarketplaceRepository
	metrics       *metrics.BusinessMetrics
	logger        *slog.Logger
	cb            *resilience.CircuitBreaker
	producer      *kafka.Writer
	categoryCache *CategoryCache
}

func NewMarketplaceService(
	repo MarketplaceRepository,
	metrics *metrics.BusinessMetrics,
	logger *slog.Logger,
	cb *resilience.CircuitBreaker,
	producer *kafka.Writer,
) *MarketplaceService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MarketplaceService{
		repo:     repo,
		metrics:  metrics,
		logger:   logger,
		cb:       cb,
		producer: producer,
		categoryCache: &CategoryCache{
			ttl: 1 * time.Hour,
		},
	}
}

func (s *MarketplaceService) GetCategories(ctx context.Context) ([]models.Category, error) {
	// Check cache first
	s.categoryCache.RLock()
	if !s.categoryCache.lastFetch.IsZero() && time.Since(s.categoryCache.lastFetch) < s.categoryCache.ttl {
		cached := make([]models.Category, len(s.categoryCache.categories))
		copy(cached, s.categoryCache.categories)
		s.categoryCache.RUnlock()
		return cached, nil
	}
	s.categoryCache.RUnlock()

	// Cache miss or expired, fetch from DB
	categories, err := s.repo.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.categoryCache.Lock()
	s.categoryCache.categories = categories
	s.categoryCache.lastFetch = time.Now()
	s.categoryCache.Unlock()

	return categories, nil
}

func (s *MarketplaceService) CreateProduct(ctx context.Context, userID primitive.ObjectID, req models.CreateProductRequest) (*models.Product, error) {
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
		Location:    models.ProductLocation{City: req.Location},
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

	s.metrics.IncrementProductsCreated()
	s.logger.Info("Product created", "product_id", createdProduct.ID, "user_id", userID)

	return createdProduct, nil
}

func (s *MarketplaceService) GetProductByID(ctx context.Context, id primitive.ObjectID, viewerID primitive.ObjectID) (*models.ProductResponse, error) {
	// Async Fire-and-Forget View Increment (FB Scale)
	if s.producer != nil {
		go func() {
			event := struct {
				ProductID string    `json:"product_id"`
				Timestamp time.Time `json:"timestamp"`
			}{
				ProductID: id.Hex(),
				Timestamp: time.Now(),
			}
			payload, _ := json.Marshal(event)

			// Non-blocking attempt (Async writer handles complexity usually, but wrapping in lightweight goroutine ensures main path speed)
			// Actually kafka-go Async writer is non-blocking on WriteMessages.
			// But creating the message struct adds micro-latency.
			// We can spawn a goroutine or just call it directly if Async=true.
			// Let's spawn safe goroutine to be ultra-safe against network stalls if buffer full.

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := s.producer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(id.Hex()),
				Value: payload,
			})
			if err != nil {
				// Don't log error on every failure in high load? Maybe debug.
				// s.logger.Debug("Failed to publish view event", "error", err)
			} else {
				// Metrics handled by consumer? Or here?
				// "Views" metric usually tracks successful DB increments.
				// "ViewEvents" tracks traffic.
				// s.metrics.IncrementProductViews() // Moved to consumer for accuracy? Or here for throughput?
				// Let's keep it here for "Attempted Views" monitoring
				s.metrics.IncrementProductViews()
			}
		}()
	}

	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Use cached categories to avoid extra DB hit
	categories, err := s.GetCategories(ctx)
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
		Seller: models.UserShortResponse{
			ID:       product.SellerID,
			Username: product.SellerUsername,
			FullName: product.SellerFullName,
			Avatar:   product.SellerAvatar,
		},
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
