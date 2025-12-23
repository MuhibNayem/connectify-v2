package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/spydotech-group/story-service/config"
	"gitlab.com/spydotech-group/story-service/internal/platform"
)

func main() {
	cfg := config.Load()

	app := platform.NewApplication(cfg)

	if err := app.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
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
		log.Fatalf("Application error: %v", err)
	}
}
