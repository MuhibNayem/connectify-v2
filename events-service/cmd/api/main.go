package main

import (
	"context"
	"log"
	"os"

	"gitlab.com/spydotech-group/events-service/config"
	"gitlab.com/spydotech-group/events-service/internal/platform"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Printf("Application error: %v", err)
		os.Exit(1)
	}
}

func run() error {
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
