package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/spydotech-group/shared-entity/observability"
	"gitlab.com/spydotech-group/story-service/config"
	"gitlab.com/spydotech-group/story-service/internal/platform"
)

func main() {
	observability.InitLogger()
	cfg := config.Load()

	app := platform.NewApplication(cfg)

	if err := app.Bootstrap(); err != nil {
		slog.Error("Failed to bootstrap application", "error", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		app.Shutdown()
		os.Exit(0)
	}()

	if err := app.Run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}
