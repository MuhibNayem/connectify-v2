package httpapi

import (
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"github.com/MuhibNayem/connectify-v2/story-service/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func BuildRouter(cfg *config.Config, handler *StoryHandler, redisClient *redis.ClusterClient) *gin.Engine {
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

	router.Use(middleware.RateLimiter(
		cfg.RateLimitEnabled,
		cfg.RateLimitLimit,
		cfg.RateLimitBurst,
		"stories:global",
		nil,
	))

	authMiddleware := middleware.AuthMiddleware(
		cfg.JWTSecret,
		redisClient.GetClient(),
		middleware.WithFailClosedResponse(http.StatusServiceUnavailable, "authentication temporarily unavailable"),
	)

	handler.RegisterRoutes(router, authMiddleware)
	return router
}
