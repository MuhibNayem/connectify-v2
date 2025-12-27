package httpapi

import (
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/marketplace-service/config"
	"github.com/MuhibNayem/connectify-v2/marketplace-service/internal/controllers"
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

	router.Use(middleware.RateLimiter(cfg.RateLimitEnabled, cfg.RateLimitLimit, cfg.RateLimitBurst, "marketplace:global", nil))

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
		// public routes
		marketplace.GET("/products", controller.SearchProducts)
		marketplace.GET("/products/:id", controller.GetProduct)
		marketplace.GET("/categories", controller.GetCategories)

		authGroup := marketplace.Group("")
		authGroup.Use(authMiddleware)
		{
			authGroup.POST("/products", controller.CreateProduct)
			authGroup.PUT("/products/:id", controller.UpdateProduct)
			authGroup.DELETE("/products/:id", controller.DeleteProduct)
			authGroup.PUT("/products/:id/sold", controller.MarkProductSold)
			authGroup.PUT("/products/:id/save", controller.ToggleSaveProduct)
			authGroup.GET("/conversations", controller.GetMarketplaceConversations)
		}
	}

	return router
}
