package httpapi

import (
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/controllers"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func BuildRouter(cfg *config.Config, marketplaceService *service.MarketplaceService, redisClient *redis.ClusterClient) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	corsCfg := cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsCfg))

	// Rate limit observer for business metrics
	var rateLimitObserver func(string)
	if businessMetrics := metrics.NewBusinessMetrics(); businessMetrics != nil {
		rateLimitObserver = businessMetrics.RecordRateLimitHit
	}

	router.Use(middleware.RateLimiter(cfg.RateLimitEnabled, cfg.RateLimitLimit, cfg.RateLimitBurst, "marketplace:global", rateLimitObserver))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	var authMiddleware gin.HandlerFunc
	if redisClient != nil {
		authMiddleware = middleware.AuthMiddleware(
			cfg.JWTSecret,
			redisClient.GetClient(),
			middleware.WithFailClosedResponse(http.StatusServiceUnavailable, "authentication temporarily unavailable"),
		)
	} else {
		authMiddleware = middleware.JWTAuthSimple(cfg.JWTSecret)
	}

	controller := controllers.NewMarketplaceController(marketplaceService)
	api := router.Group("/api/v1")

	marketplace := api.Group("/marketplace")
	{
		// Public routes with appropriate rate limits
		marketplace.GET("/products", 
			middleware.StrictRateLimiter(5, 20, "marketplace:search", rateLimitObserver),
			controller.SearchProducts,
		)
		marketplace.GET("/products/:id", 
			middleware.StrictRateLimiter(10, 30, "marketplace:view", rateLimitObserver),
			controller.GetProduct,
		)
		marketplace.GET("/categories", controller.GetCategories)

		authGroup := marketplace.Group("")
		authGroup.Use(authMiddleware)
		{
			authGroup.POST("/products", 
				middleware.StrictRateLimiter(0.1, 3, "marketplace:create", rateLimitObserver), // 6 per minute
				controller.CreateProduct,
			)
			authGroup.PUT("/products/:id", 
				middleware.StrictRateLimiter(0.5, 5, "marketplace:update", rateLimitObserver), // 30 per minute
				controller.UpdateProduct,
			)
			authGroup.DELETE("/products/:id", 
				middleware.StrictRateLimiter(0.2, 2, "marketplace:delete", rateLimitObserver), // 12 per minute
				controller.DeleteProduct,
			)
			authGroup.PUT("/products/:id/sold", 
				middleware.StrictRateLimiter(0.5, 5, "marketplace:sold", rateLimitObserver),
				controller.MarkProductSold,
			)
			authGroup.PUT("/products/:id/save", 
				middleware.StrictRateLimiter(2, 10, "marketplace:save", rateLimitObserver), // 120 per minute
				controller.ToggleSaveProduct,
			)
			authGroup.GET("/conversations", 
				middleware.StrictRateLimiter(1, 5, "marketplace:conversations", rateLimitObserver),
				controller.GetMarketplaceConversations,
			)
		}
	}

	return router
}
