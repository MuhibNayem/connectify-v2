package main

import (
	"context"
	"log"
	"time"

	"community-service/internal/config"
	"community-service/internal/grpc"
	"community-service/internal/repository"
	"community-service/internal/service"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	cfg := config.LoadConfig()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to MongoDB")

	db := client.Database(cfg.DBName)

	// Init Layers
	communityRepo := repository.NewCommunityRepository(db)
	communityService := service.NewCommunityService(communityRepo)
	grpcServer := grpc.NewServer(communityService)

	// Start Server
	log.Printf("Starting Community Service on port %s", cfg.GRPCPort)
	if err := grpcServer.Start(cfg.GRPCPort); err != nil {
		log.Fatal(err)
	}
}
