package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"gitlab.com/spydotech-group/events-service/config"
	"gitlab.com/spydotech-group/events-service/internal/platform"
	"gitlab.com/spydotech-group/shared-entity/observability"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	observability.InitLogger()

	// Load .env file if it exists
	_ = godotenv.Load()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize metrics
	metrics := config.GetMetrics()

	// Create and run application
	app, err := platform.NewApplication(context.Background(), cfg, metrics)
	if err != nil {
		return err
	}

	return app.Run()
}
