package platform

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Dependencies struct {
	Config             *config.Config
	MongoDB            *mongo.Database
	MarketplaceRepo    *repository.MarketplaceRepository
	MarketplaceService *service.MarketplaceService
}

func InitializeDependencies(cfg *config.Config) (*Dependencies, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize Tracer
	tp, err := observability.InitTracer(context.Background(), observability.TracerConfig{
		ServiceName:    "marketplace-service",
		ServiceVersion: "1.0.0",
		Environment:    "development", // TODO: Configurable
		JaegerEndpoint: cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	}
	_ = tp

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = mongoClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	slog.Info("Connected to MongoDB")

	mongoDB := mongoClient.Database("messaging_app")
	marketplaceRepo := repository.NewMarketplaceRepository(mongoDB)
	marketplaceService := service.NewMarketplaceService(marketplaceRepo)

	return &Dependencies{
		Config:             cfg,
		MongoDB:            mongoDB,
		MarketplaceRepo:    marketplaceRepo,
		MarketplaceService: marketplaceService,
	}, nil
}
