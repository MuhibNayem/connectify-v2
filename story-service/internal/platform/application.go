package platform

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"gitlab.com/spydotech-group/story-service/config"
	storygrpc "gitlab.com/spydotech-group/story-service/internal/grpc"
	"gitlab.com/spydotech-group/story-service/internal/producer"
	"gitlab.com/spydotech-group/story-service/internal/repository"
	"gitlab.com/spydotech-group/story-service/internal/service"
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

	clientOptions := options.Client().ApplyURI(a.cfg.MongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	a.mongoClient = mongoClient
	log.Println("Connected to MongoDB")

	db := mongoClient.Database(a.cfg.DBName)

	// Initialize Kafka producer
	a.producer = producer.NewStoryProducer(a.cfg.KafkaBrokers, a.cfg.KafkaTopic)
	log.Println("Kafka producer initialized")

	// Initialize repositories
	a.storyRepo = repository.NewStoryRepository(db)

	// Initialize services
	a.storyService = service.NewStoryService(a.storyRepo, a.producer)

	// Initialize gRPC server
	a.grpcServer = grpc.NewServer()
	a.grpcHandler = storygrpc.NewServer(a.storyService)
	a.grpcHandler.Register(a.grpcServer)

	log.Println("Application bootstrapped successfully")
	return nil
}

func (a *Application) Run() error {
	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", a.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", a.cfg.GRPCPort, err)
	}

	log.Printf("Story service gRPC server listening on :%s", a.cfg.GRPCPort)

	if err := a.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

func (a *Application) Shutdown() {
	log.Println("Shutting down story-service...")

	// Stop gRPC server
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
		log.Println("gRPC server stopped")
	}

	// Close Kafka producer
	if a.producer != nil {
		if err := a.producer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		} else {
			log.Println("Kafka producer closed")
		}
	}

	// Disconnect MongoDB
	if a.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.mongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("MongoDB disconnected")
		}
	}

	log.Println("Story service shutdown complete")
}
