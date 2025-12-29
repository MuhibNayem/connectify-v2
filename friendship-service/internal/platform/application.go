package platform

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MuhibNayem/connectify-v2/friendship-service/config"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/cache"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/graph"
	internalgrpc "github.com/MuhibNayem/connectify-v2/friendship-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/httpapi"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/kafka"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/outbox"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/reconciliation"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/validation"

	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	friendshippb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/friendship/v1"
	pkgredis "github.com/MuhibNayem/connectify-v2/shared-entity/redis"

	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Application holds all application dependencies
type Application struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg             *config.Config
	businessMetrics *metrics.BusinessMetrics

	mongoClient *mongo.Client
	db          *mongo.Database
	redisClient *pkgredis.ClusterClient
	neo4jClient *graph.Neo4jClient

	kafkaProducer *kafka.MessageProducer

	// Consistency components
	outboxRepo      *outbox.Repository
	outboxProcessor *outbox.Processor
	cachePubSub     *cache.PubSub
	reconcileJob    *reconciliation.Job
	graphRepo       *repository.GraphRepository

	friendshipService *service.FriendshipService
	grpcServer        *grpc.Server
	httpServer        *http.Server
	metricsServer     *http.Server

	shutdownOnce sync.Once
}

// NewApplication creates a new Application instance
func NewApplication(parentCtx context.Context, cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	app := &Application{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}

	if err := app.bootstrap(); err != nil {
		app.Close()
		return nil, err
	}
	return app, nil
}

// Run starts all servers and waits for shutdown signal
func (a *Application) Run() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 3)

	// Start HTTP server
	if a.httpServer != nil {
		go func() {
			slog.Info("HTTP server starting", "address", a.httpServer.Addr)
			if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTP server failed: %w", err)
			}
		}()
	}

	// Start metrics server
	go func() {
		slog.Info("Metrics server starting", "address", a.metricsServer.Addr)
		if err := a.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", net.JoinHostPort("", a.cfg.GRPCPort))
		if err != nil {
			errCh <- fmt.Errorf("failed to listen for gRPC: %w", err)
			return
		}
		slog.Info("gRPC server starting", "address", lis.Addr())
		if err := a.grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()

	// Start outbox processor (background worker for eventual consistency)
	if a.outboxProcessor != nil {
		go a.outboxProcessor.Start(a.ctx)
	}

	// Start cache invalidation pub/sub
	if a.cachePubSub != nil {
		a.cachePubSub.Subscribe(a.ctx)
	}

	// Start reconciliation job
	if a.reconcileJob != nil {
		go a.reconcileJob.Start(a.ctx)
	}

	select {
	case <-quit:
		slog.Info("Received shutdown signal")
		return a.Shutdown()
	case err := <-errCh:
		slog.Error("Server error", "error", err)
		return a.Shutdown()
	}
}

// Shutdown gracefully stops the application
func (a *Application) Shutdown() error {
	var shutdownErr error
	a.shutdownOnce.Do(func() {
		slog.Info("Shutting down application...")
		a.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if a.httpServer != nil {
			if err := a.httpServer.Shutdown(ctx); err != nil {
				slog.Error("HTTP server shutdown error", "error", err)
				shutdownErr = err
			}
		}

		if a.metricsServer != nil {
			if err := a.metricsServer.Shutdown(ctx); err != nil {
				slog.Error("Metrics server shutdown error", "error", err)
				shutdownErr = err
			}
		}

		if a.grpcServer != nil {
			slog.Info("Stopping gRPC server...")
			a.grpcServer.GracefulStop()
		}

		a.Close()
	})
	return shutdownErr
}

// Close releases all resources
func (a *Application) Close() {
	if a.kafkaProducer != nil {
		a.kafkaProducer.Close()
	}
	if a.neo4jClient != nil {
		_ = a.neo4jClient.Close(context.Background())
	}
	if a.redisClient != nil {
		_ = a.redisClient.Close()
	}
	if a.mongoClient != nil {
		_ = a.mongoClient.Disconnect(context.Background())
	}
}

func (a *Application) bootstrap() error {
	var err error

	// Initialize tracer
	tp, err := observability.InitTracer(a.ctx, observability.TracerConfig{
		ServiceName:    "friendship-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		JaegerEndpoint: a.cfg.JaegerOTLPEndpoint,
	})
	if err != nil {
		slog.Warn("Failed to initialize tracer", "error", err)
	}
	_ = tp

	// Initialize MongoDB
	a.mongoClient, a.db, err = InitMongo(a.ctx, a.cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Initialize Redis
	a.redisClient, err = InitRedis(a.cfg)
	if err != nil {
		slog.Warn("Failed to connect to Redis", "error", err)
	}

	// Initialize Neo4j
	a.neo4jClient, err = InitNeo4j(a.cfg)
	if err != nil {
		slog.Warn("Failed to connect to Neo4j", "error", err)
	}

	// Initialize Kafka producer
	a.kafkaProducer = kafka.NewMessageProducer(a.cfg.KafkaBrokers, a.cfg.KafkaTopic)

	// Initialize components
	logger := slog.Default()
	a.businessMetrics = metrics.NewBusinessMetrics()
	validator := validation.NewValidator()
	circuitBreaker := service.NewCircuitBreakerWrapper(logger)

	// Initialize repositories
	friendshipRepo := repository.NewFriendshipRepository(a.db)
	if a.neo4jClient != nil {
		a.graphRepo = repository.NewGraphRepository(a.neo4jClient.Driver)
	}

	// Initialize user client
	userClient, err := repository.NewUserClient(a.cfg.UserServiceURL)
	if err != nil {
		return fmt.Errorf("failed to connect to user service: %w", err)
	}

	// Initialize cache
	var friendshipCache *cache.FriendshipCache
	if a.redisClient != nil {
		friendshipCache = cache.NewFriendshipCache(a.redisClient, a.cfg.CacheTTL)

		// Initialize cache pub/sub for distributed invalidation
		a.cachePubSub = cache.NewPubSub(a.redisClient, friendshipCache, logger)
	}

	// Initialize outbox for eventual consistency
	a.outboxRepo = outbox.NewRepository(a.db)
	a.outboxProcessor = outbox.NewProcessor(
		a.outboxRepo,
		a.graphRepo,
		a.kafkaProducer,
		logger,
	)

	// Initialize reconciliation job
	a.reconcileJob = reconciliation.NewJob(
		friendshipRepo,
		a.graphRepo,
		friendshipCache,
		a.businessMetrics,
		logger,
	)

	// Initialize service
	a.friendshipService = service.NewFriendshipService(
		friendshipRepo,
		userClient,
		a.graphRepo,
		a.kafkaProducer,
		friendshipCache,
		circuitBreaker,
		a.businessMetrics,
		validator,
		logger,
	)

	// Initialize gRPC server
	a.grpcServer = grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	grpcHandler := internalgrpc.NewFriendshipServer(a.friendshipService)
	friendshippb.RegisterFriendshipServiceServer(a.grpcServer, grpcHandler)
	reflection.Register(a.grpcServer)

	// Initialize HTTP server with protected API
	httpHandler := httpapi.NewFriendshipHandler(a.friendshipService)
	router := httpapi.BuildRouter(a.cfg, httpHandler, a.redisClient)
	a.httpServer = &http.Server{
		Addr:    net.JoinHostPort("", a.cfg.HTTPPort),
		Handler: router,
	}

	// Initialize metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", config.MetricsHandler())
	a.metricsServer = &http.Server{
		Addr:    net.JoinHostPort("", a.cfg.PrometheusPort),
		Handler: metricsMux,
	}

	return nil
}
