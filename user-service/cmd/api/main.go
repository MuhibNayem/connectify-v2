package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"user-service/config"
	"user-service/internal/events"
	grpchandler "user-service/internal/handler/grpc"
	httphandler "user-service/internal/handler/http"
	"user-service/internal/repository"
	"user-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gitlab.com/spydotech-group/shared-entity/observability"
	pb "gitlab.com/spydotech-group/shared-entity/proto/user/v1"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	observability.InitLogger()
	var cfg *config.Config = config.LoadConfig()

	// 0. Observability
	tp, err := observability.InitTracer(context.Background(), observability.TracerConfig{
		ServiceName:    "user-service",
		ServiceVersion: "1.0.0",
		Environment:    "development", // TODO: Make configurable
		JaegerEndpoint: cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				slog.Error("Error shutting down tracer provider", "error", err)
			}
		}()
	}

	// 1. Database Connections
	// Mongo
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		slog.Error("Mongo connect error", "error", err)
		os.Exit(1)
	}
	db := mongoClient.Database(cfg.DBName)

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURLs[0],
		Password: cfg.RedisPass,
	})

	// Neo4j
	neoDriver, err := neo4j.NewDriverWithContext(cfg.Neo4jURI, neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""))
	if err != nil {
		slog.Error("Neo4j connect error", "error", err)
		os.Exit(1)
	}
	defer neoDriver.Close(context.Background())

	// 2. Repositories
	userRepo := repository.NewUserRepository(db)
	graphRepo := repository.NewGraphRepository(neoDriver)
	// friendRepo := repository.NewFriendshipRepository(db)

	// 3. Producers
	producer := events.NewEventProducer(cfg.KafkaBrokers, cfg.UserUpdatedTopic)
	defer producer.Close()

	// 4. Services
	authService := service.NewAuthService(userRepo, graphRepo, redisClient, cfg)
	userService := service.NewUserService(userRepo, producer, redisClient, cfg)
	// friendshipService := service.NewFriendshipService(friendRepo, graphRepo, userRepo, producer, cfg) // Unused in handlers yet

	// 5. Handlers
	authHandler := httphandler.NewAuthHandler(authService)
	userGrpcHandler := grpchandler.NewUserHandler(userService)

	// 6. Servers
	// HTTP Server (Gin)
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user-service"}) })

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}
	}

	go func() {
		slog.Info("Starting HTTP server", "port", cfg.ServerPort)
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// gRPC Server
	grpcServer := grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	pb.RegisterUserServiceServer(grpcServer, userGrpcHandler)
	reflection.Register(grpcServer)

	// gRPC Port
	grpcPort := "9083"
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		slog.Error("Failed to listen for gRPC", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Starting gRPC server", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down...")
}
