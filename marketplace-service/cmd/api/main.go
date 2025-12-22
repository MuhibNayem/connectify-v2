package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"gitlab.com/spydotech-group/marketplace-service/config"
	grpcserver "gitlab.com/spydotech-group/marketplace-service/internal/grpc"
	"gitlab.com/spydotech-group/marketplace-service/internal/platform"
	marketplacepb "gitlab.com/spydotech-group/shared-entity/proto/marketplace/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Application error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg := config.LoadConfig()

	// Initialize dependencies
	deps, err := platform.InitializeDependencies(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	// Create gRPC server
	grpcSrv := grpc.NewServer()
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
		log.Printf("🚀 Marketplace gRPC Service listening on port %s", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-done
	log.Println("📴 Shutting down gracefully...")
	grpcSrv.GracefulStop()

	return nil
}
