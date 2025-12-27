package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/MuhibNayem/connectify-v2/feed-service/internal/config"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/events"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/graph"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/feed-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to Mongo: %v", err)
	}
	db := client.Database(cfg.DBName)

	// Initialize Layers
	repo := repository.NewFeedRepository(db)

	// Ensure Indexes
	if err := repo.EnsureIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to ensure indexes: %v", err)
	}

	// Initialize Neo4j Client
	neo4jClient, err := graph.NewNeo4jClient(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jClient.Close(context.Background())

	graphRepo := repository.NewGraphRepository(neo4jClient.Driver)

	// Initialize Redis Cache
	cacheRepo := repository.NewCacheRepository(cfg.RedisAddrs, cfg.RedisPassword)

	// Start Event Listener
	eventListener := events.NewEventListener(cfg, repo, cacheRepo, graphRepo)
	ctxBg := context.Background()
	eventListener.Start(ctxBg)

	// Event Producer
	producer := events.NewEventProducer(cfg)
	defer producer.Close()

	svc := service.NewFeedService(repo, cacheRepo, graphRepo, producer)
	handler := grpc.NewServer(svc)

	// Start gRPC Server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := googlegrpc.NewServer()
	handler.Register(grpcServer)

	log.Printf("Feed Service listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
