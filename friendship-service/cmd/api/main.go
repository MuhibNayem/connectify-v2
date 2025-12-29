package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/MuhibNayem/connectify-v2/friendship-service/config"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/platform"
)

func main() {
	// Initialize structured logger (using slog)
	slog.SetDefault(slog.Default())

	slog.Info("Starting Friendship Service")

	cfg := config.LoadConfig()

	app, err := platform.NewApplication(context.Background(), cfg)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}
