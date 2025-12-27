package main

import (
	"context"
	"errors"
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
	"user-service/internal/platform"
	"user-service/internal/repository"
	"user-service/internal/service"

	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	pb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/gin-gonic/gin"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	observability.InitLogger()
	if err := run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.LoadConfig()

	tp, err := observability.InitTracer(ctx, observability.TracerConfig{
		ServiceName:    "user-service",
		ServiceVersion: "1.0.0",
		Environment:    "development", // TODO: Make configurable
		JaegerEndpoint: cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	}
	if tp != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tp.Shutdown(shutdownCtx); err != nil {
				slog.Error("Error shutting down tracer provider", "error", err)
			}
		}()
	}

	// 1. Database Connections
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(dbCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return err
	}
	db := mongoClient.Database(cfg.DBName)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURLs[0],
		Password: cfg.RedisPass,
	})

	neoDriver, err := neo4j.NewDriverWithContext(cfg.Neo4jURI, neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""))
	if err != nil {
		return err
	}

	// 2. Repositories
	userRepo := repository.NewUserRepository(db)
	graphRepo := repository.NewGraphRepository(neoDriver)

	// 3. Producers
	producer := events.NewEventProducer(cfg.KafkaBrokers, cfg.UserUpdatedTopic, slog.Default())

	// 4. Business Metrics
	businessMetrics := platform.NewBusinessMetrics()

	// 5. Services
	authService := service.NewAuthService(userRepo, graphRepo, redisClient, cfg)
	userService := service.NewUserService(userRepo, producer, redisClient, cfg, slog.Default(), businessMetrics)
	rateLimitObserver := businessMetrics.RecordRateLimitHit

	// 5. Handlers
	authHandler := httphandler.NewAuthHandler(authService, cfg)
	userHandler := httphandler.NewUserHandler(userService)
	userGrpcHandler := grpchandler.NewUserHandler(userService)

	// HTTP Server
	r := gin.Default()
	r.Use(middleware.RateLimiter(
		cfg.RateLimitEnabled,
		cfg.RateLimitLimit,
		cfg.RateLimitBurst,
		"user:global",
		rateLimitObserver,
	))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "user-service"})
	})
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register",
				middleware.StrictRateLimiter(0.1, 3, "auth:register", rateLimitObserver), // ≈6 requests/min
				authHandler.Register,
			)
			auth.POST("/login",
				middleware.StrictRateLimiter(1, 8, "auth:login", rateLimitObserver), // limit brute force attempts
				authHandler.Login,
			)
			auth.POST("/refresh",
				middleware.StrictRateLimiter(0.5, 5, "auth:refresh", rateLimitObserver),
				authHandler.RefreshToken,
			)
		}

		// User routes (public profile endpoints)
		users := api.Group("/users")
		{
			users.GET("/:id", userHandler.GetUserByID)
			users.GET("/:id/status", userHandler.GetUserStatus)
		}
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	grpcServer := grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	pb.RegisterUserServiceServer(grpcServer, userGrpcHandler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":9083")
	if err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() {
		slog.Info("Starting HTTP server", "port", cfg.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		slog.Info("Starting gRPC server", "port", "9083")
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutdown signal received")
	case err := <-errCh:
		if err != nil {
			slog.Error("Server error", "error", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}
	grpcServer.GracefulStop()

	if err := mongoClient.Disconnect(shutdownCtx); err != nil {
		slog.Error("Mongo disconnect error", "error", err)
	}
	if err := redisClient.Close(); err != nil {
		slog.Error("Redis close error", "error", err)
	}
	if err := neoDriver.Close(shutdownCtx); err != nil {
		slog.Error("Neo4j close error", "error", err)
	}
	if err := producer.Close(); err != nil {
		slog.Error("Kafka producer close error", "error", err)
	}

	return nil
}
