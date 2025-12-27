package platform

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/controllers"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/repository"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/resilience"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type Application struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.Config

	mongoClient *mongo.Client
	db          *mongo.Database

	marketplaceService *service.MarketplaceService
	mainRouter         *gin.Engine
	httpServer         *http.Server
	metricsServer      *http.Server

	metrics *metrics.BusinessMetrics
	logger  *slog.Logger

	shutdownOnce sync.Once
}

func NewApplication(parentCtx context.Context, cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	app := &Application{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		logger: slog.Default(),
	}

	if err := app.initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize application: %w", err)
	}

	return app, nil
}

func (app *Application) initialize() error {
	if err := app.initializeDatabase(); err != nil {
		return err
	}

	app.metrics = metrics.NewBusinessMetrics()

	if err := app.initializeServices(); err != nil {
		return err
	}

	app.setupRouter()

	return nil
}

func (app *Application) initializeDatabase() error {
	deps, err := InitializeDependencies(app.cfg)
	if err != nil {
		return err
	}

	app.mongoClient = nil // Store if needed
	app.db = deps.MongoDB

	return nil
}

func (app *Application) initializeServices() error {
	marketplaceRepo := repository.NewMarketplaceRepository(app.db)

	cb := resilience.NewCircuitBreaker(resilience.DefaultConfig("marketplace-db"), app.logger)

	app.marketplaceService = service.NewMarketplaceService(
		marketplaceRepo,
		app.metrics,
		app.logger,
		cb,
	)

	app.logger.Info("✅ Services initialized")
	return nil
}

func (app *Application) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	// Use structured logging middleware here in future
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		marketplace := api.Group("/marketplace")
		{
			controller := controllers.NewMarketplaceController(app.marketplaceService)
			marketplace.POST("/products", controller.CreateProduct)
			marketplace.GET("/products/:id", controller.GetProduct)
			marketplace.GET("/products", controller.SearchProducts)
			marketplace.PUT("/products/:id", controller.UpdateProduct)
			marketplace.DELETE("/products/:id", controller.DeleteProduct)
			marketplace.PUT("/products/:id/sold", controller.MarkProductSold)
			marketplace.PUT("/products/:id/save", controller.ToggleSaveProduct)

			marketplace.GET("/categories", controller.GetCategories)
			marketplace.GET("/conversations", controller.GetMarketplaceConversations)
		}
	}

	app.mainRouter = router
	app.logger.Info("✅ Router configured")
}

func (app *Application) Run() error {
	app.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", app.cfg.GRPCPort),
		Handler: app.mainRouter,
	}

	// Setup metrics server
	app.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", app.cfg.MetricsPort),
		Handler: promhttp.Handler(),
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("✅ Metrics server listening on port %s", app.cfg.MetricsPort)
		if err := app.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Start main HTTP server
	go func() {
		log.Printf("🚀 Marketplace Service listening on port %s", app.cfg.GRPCPort)
		if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-shutdownChan
	log.Println("📴 Shutdown signal received...")

	return app.Shutdown()
}

func (app *Application) Shutdown() error {
	var shutdownErr error

	app.shutdownOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown HTTP servers
		if app.httpServer != nil {
			if err := app.httpServer.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown error: %v", err)
				shutdownErr = err
			}
		}

		if app.metricsServer != nil {
			if err := app.metricsServer.Shutdown(ctx); err != nil {
				log.Printf("Metrics server shutdown error: %v", err)
			}
		}

		// Close database connections
		if app.mongoClient != nil {
			if err := app.mongoClient.Disconnect(ctx); err != nil {
				log.Printf("MongoDB disconnect error: %v", err)
			}
		}

		app.cancel()
		log.Println("✅ Application shutdown complete")
	})

	return shutdownErr
}
