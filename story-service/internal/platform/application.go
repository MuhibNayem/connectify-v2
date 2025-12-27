package platform

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	"github.com/MuhibNayem/connectify-v2/story-service/config"
	storygrpc "github.com/MuhibNayem/connectify-v2/story-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

type Application struct {
	cfg         *config.Config
	mongoClient *mongo.Client
	grpcServer  *grpc.Server
	producer    *producer.StoryProducer

	// Repositories
	storyRepo *repository.StoryRepository

	// Services
	storyService *service.StoryService

	// gRPC Server
	grpcHandler *storygrpc.Server
}

func NewApplication(cfg *config.Config) *Application {
	return &Application{cfg: cfg}
}

func (a *Application) Bootstrap() error {
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize Tracer
	tp, err := observability.InitTracer(context.Background(), observability.TracerConfig{
		ServiceName:    "story-service",
		ServiceVersion: "1.0.0",
		Environment:    "development", // TODO: Configurable
		JaegerEndpoint: a.cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	}
	_ = tp // Keep reference if needed later, but suppress error for now
	// Note: We are not deferring shutdown here because Bootstrap exits.
	// We should add a Close method or handle it in Shutdown.

	clientOptions := options.Client().ApplyURI(a.cfg.MongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	a.mongoClient = mongoClient
	slog.Info("Connected to MongoDB")

	db := mongoClient.Database(a.cfg.DBName)

	// Initialize Kafka producer
	a.producer = producer.NewStoryProducer(a.cfg.KafkaBrokers, a.cfg.KafkaTopic)
	slog.Info("Kafka producer initialized")

	// Initialize repositories
	a.storyRepo = repository.NewStoryRepository(db)

	// Initialize services
	a.storyService = service.NewStoryService(a.storyRepo, a.producer)

	// Initialize gRPC server
	a.grpcServer = grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	a.grpcHandler = storygrpc.NewServer(a.storyService)
	a.grpcHandler.Register(a.grpcServer)

	slog.Info("Application bootstrapped successfully")
	return nil
}

func (a *Application) Run() error {
	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", a.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", a.cfg.GRPCPort, err)
	}

	slog.Info("Story service gRPC server listening", "port", a.cfg.GRPCPort)

	if err := a.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

func (a *Application) Shutdown() {
	slog.Info("Shutting down story-service...")

	// Stop gRPC server
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
		slog.Info("gRPC server stopped")
	}

	// Close Kafka producer
	if a.producer != nil {
		if err := a.producer.Close(); err != nil {
			slog.Error("Error closing Kafka producer", "error", err)
		} else {
			slog.Info("Kafka producer closed")
		}
	}

	// Disconnect MongoDB
	if a.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.mongoClient.Disconnect(ctx); err != nil {
			slog.Error("Error disconnecting from MongoDB", "error", err)
		} else {
			slog.Info("MongoDB disconnected")
		}
	}

	slog.Info("Story service shutdown complete")
}
