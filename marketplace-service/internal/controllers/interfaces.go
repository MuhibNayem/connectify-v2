package controllers

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MarketplaceService defines the interface for marketplace operations
type MarketplaceService interface {
	GetCategories(ctx context.Context) ([]models.Category, error)
	CreateProduct(ctx context.Context, userID primitive.ObjectID, req models.CreateProductRequest) (*models.Product, error)
	GetProductByID(ctx context.Context, id primitive.ObjectID, viewerID primitive.ObjectID) (*models.ProductResponse, error)
	SearchProducts(ctx context.Context, filter models.ProductFilter) (*service.MarketplaceListResponse, error)
	GetMarketplaceConversations(ctx context.Context, userID primitive.ObjectID) ([]models.ConversationSummary, error)
	MarkProductSold(ctx context.Context, productID, userID primitive.ObjectID) error
	DeleteProduct(ctx context.Context, productID, userID primitive.ObjectID) error
	ToggleSaveProduct(ctx context.Context, productID, userID primitive.ObjectID) (bool, error)
}
