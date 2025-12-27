package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/reel-service/config"
	reelgrpc "github.com/MuhibNayem/connectify-v2/reel-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/httpapi"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	userpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Application struct {
	cfg         *config.Config
	mongoClient *mongo.Client
	grpcServer  *grpc.Server
	producer    *producer.ReelProducer
	httpServer  *http.Server
	redisClient *redis.ClusterClient
	userConn    *grpc.ClientConn
	userClient  userpb.UserServiceClient

	reelRepo    *repository.ReelRepository
	reelService *service.ReelService
	grpcHandler *reelgrpc.Server
}

func NewApplication(cfg *config.Config) *Application {
	return &Application{cfg: cfg}
}

func (a *Application) Bootstrap() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tp, err := observability.InitTracer(context.Background(), observability.TracerConfig{
		ServiceName:    "reel-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		JaegerEndpoint: a.cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	}
	_ = tp

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

	a.producer = producer.NewReelProducer(a.cfg.KafkaBrokers, a.cfg.KafkaTopic)
	slog.Info("Kafka producer initialized")

	a.reelRepo = repository.NewReelRepository(db)

	a.grpcServer = grpc.NewServer(
		observability.GetGRPCServerOption(),
	)

	if err := a.initRedis(); err != nil {
		return fmt.Errorf("failed to initialize redis: %w", err)
	}

	if err := a.initUserClient(); err != nil {
		return fmt.Errorf("failed to connect to user service: %w", err)
	}

	businessMetrics := metrics.NewBusinessMetrics()

	circuitBreaker := resilience.NewCircuitBreaker(
		resilience.DefaultConfig("user-service"),
		slog.Default(),
	)

	a.reelService = service.NewReelService(
		a.reelRepo,
		a.producer,
		a.userClient,
		circuitBreaker,
		businessMetrics,
		slog.Default(),
		a.redisClient,
	)

	a.grpcHandler = reelgrpc.NewServer(a.reelService)
	a.grpcHandler.Register(a.grpcServer)

	httpHandler := httpapi.NewReelHandler(a.reelService, a.userClient, businessMetrics)
	router := httpapi.BuildRouter(a.cfg, httpHandler, a.redisClient)
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.cfg.ServerPort),
		Handler: router,
	}

	slog.Info("Application bootstrapped successfully")
	return nil
}

func (a *Application) Run() error {
	errCh := make(chan error, 2)

	if a.httpServer != nil {
		go func() {
			slog.Info("Reel service HTTP server listening", "port", a.cfg.ServerPort)
			if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}

	go func() {
		errCh <- a.startGRPC()
	}()

	return <-errCh
}

func (a *Application) startGRPC() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", a.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", a.cfg.GRPCPort, err)
	}

	slog.Info("Reel service gRPC server listening", "port", a.cfg.GRPCPort)

	if err := a.grpcServer.Serve(lis); err != nil {
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

func (a *Application) Shutdown() {
	slog.Info("Shutting down reel-service...")

	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down HTTP server", "error", err)
		} else {
			slog.Info("HTTP server stopped")
		}
	}

	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
		slog.Info("gRPC server stopped")
	}

	if a.producer != nil {
		if err := a.producer.Close(); err != nil {
			slog.Error("Error closing Kafka producer", "error", err)
		} else {
			slog.Info("Kafka producer closed")
		}
	}

	if a.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.mongoClient.Disconnect(ctx); err != nil {
			slog.Error("Error disconnecting from MongoDB", "error", err)
		} else {
			slog.Info("MongoDB disconnected")
		}
	}

	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			slog.Error("Error closing Redis connection", "error", err)
		} else {
			slog.Info("Redis client closed")
		}
	}

	if a.userConn != nil {
		if err := a.userConn.Close(); err != nil {
			slog.Error("Error closing user-service client connection", "error", err)
		} else {
			slog.Info("User-service client connection closed")
		}
	}

	slog.Info("Reel service shutdown complete")
}

func (a *Application) initRedis() error {
	cfg := redis.Config{
		RedisURLs: a.cfg.RedisURLs,
		RedisPass: a.cfg.RedisPass,
	}
	client := redis.NewClusterClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to connect to Redis cluster within timeout")
		case <-ticker.C:
			if client.IsAvailable(context.Background()) {
				a.redisClient = client
				slog.Info("Connected to Redis cluster")
				return nil
			}
			slog.Warn("Waiting for Redis cluster...")
		}
	}
}

func (a *Application) initUserClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx,
		net.JoinHostPort(a.cfg.UserServiceHost, a.cfg.UserServicePort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	a.userConn = conn
	a.userClient = userpb.NewUserServiceClient(conn)
	slog.Info("Connected to user service", "host", a.cfg.UserServiceHost, "port", a.cfg.UserServicePort)
	return nil
}
