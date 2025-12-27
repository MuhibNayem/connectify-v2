package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MuhibNayem/connectify-v2/reel-service/config"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/platform"
	"github.com/MuhibNayem/connectify-v2/shared-entity/observability"
)

func main() {
	observability.InitLogger()
	cfg := config.Load()

	app := platform.NewApplication(cfg)

	if err := app.Bootstrap(); err != nil {
		slog.Error("Failed to bootstrap application", "error", err)
		os.Exit(1)
	}

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
