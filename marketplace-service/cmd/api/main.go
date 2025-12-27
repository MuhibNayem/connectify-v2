package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	grpcserver "github.com/MuhibNayem/connectify-v2/marketplace-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/platform"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	marketplacepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/marketplace/v1"
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

	// Create gRPC server
	grpcSrv := grpc.NewServer(
		observability.GetGRPCServerOption(),
	)
	marketplacepb.RegisterMarketplaceServiceServer(grpcSrv, grpcserver.NewServer(deps.MarketplaceService))

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Marketplace gRPC Service listening", "port", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("Shutting down gracefully...")
	grpcSrv.GracefulStop()

	return nil
}
