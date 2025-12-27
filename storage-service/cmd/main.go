package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	storagepb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/storage/v1"
	"github.com/MuhibNayem/connectify-v2/storage-service/config"
	grpchandler "github.com/MuhibNayem/connectify-v2/storage-service/internal/grpc"
	"github.com/MuhibNayem/connectify-v2/storage-service/internal/httpapi"
	"github.com/MuhibNayem/connectify-v2/storage-service/internal/service"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	storageSvc, err := service.NewStorageService(cfg, logger)
	if err != nil {
		log.Fatalf("Failed to create storage service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startGRPCServer(cfg, storageSvc, logger)
	go startHTTPServer(cfg, storageSvc, logger)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	cancel()
	_ = ctx
}

func startGRPCServer(cfg *config.Config, svc *service.StorageService, logger *slog.Logger) {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer()
	storagepb.RegisterStorageServiceServer(grpcServer, grpchandler.NewStorageHandler(svc))

	logger.Info("gRPC server starting", "port", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, svc *service.StorageService, logger *slog.Logger) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	handler := httpapi.NewStorageHandler(svc)
	handler.RegisterRoutes(r)

	logger.Info("HTTP server starting", "port", cfg.HTTPPort)
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
