package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MuhibNayem/connectify-v2/events-service/config"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/platform"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
	"github.com/joho/godotenv"
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

	metrics := config.GetMetrics()
	app, err := platform.NewApplication(context.Background(), cfg, metrics)
	if err != nil {
		return err
	}

	return app.Run()
}
