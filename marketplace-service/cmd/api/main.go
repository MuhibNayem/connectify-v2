package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	grpcserver "github.com/MuhibNayem/connectify-v2/marketplace-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/httpapi"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/platform"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	marketplacepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/marketplace/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	observability.InitLogger()
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	// Initialize dependencies
	deps, err := platform.InitializeDependencies(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	redisClient, err := initRedis(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize redis: %w", err)
	}

	httpRouter := httpapi.BuildRouter(cfg, deps.MarketplaceService, redisClient)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: httpRouter,
	}

	// Create gRPC server
	grpcSrv := grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	marketplacepb.RegisterMarketplaceServiceServer(grpcSrv, grpcserver.NewServer(deps.MarketplaceService))

	// Setup metrics server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.MetricsPort),
		Handler: promhttp.Handler(),
	}

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Marketplace HTTP Service listening", "port", cfg.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			done <- syscall.SIGTERM
		}
	}()

	go func() {
		slog.Info("Marketplace Metrics Service listening", "port", cfg.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server error", "error", err)
		}
	}()

	go func() {
		slog.Info("Marketplace gRPC Service listening", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	if metricsServer != nil {
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Metrics server shutdown error", "error", err)
		}
	}

	grpcSrv.GracefulStop()
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			slog.Error("Redis close error", "error", err)
		}
	}

	return nil
}

func initRedis(cfg *config.Config) (*redis.ClusterClient, error) {
	client := redis.NewClusterClient(redis.Config{
		RedisURLs: cfg.RedisURLs,
		RedisPass: cfg.RedisPass,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("redis cluster not available")
		case <-ticker.C:
			if client.IsAvailable(context.Background()) {
				slog.Info("Connected to Redis cluster")
				return client, nil
			}
			slog.Warn("Waiting for Redis cluster...")
		}
	}
}
