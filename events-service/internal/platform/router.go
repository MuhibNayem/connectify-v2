package platform

import (
	"context"
	"net/http"
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/config"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/controllers"

	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

type RouterConfig struct {
	EventController *controllers.EventController
}

func (a *Application) buildRouters(cfg RouterConfig) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(config.MetricsMiddleware(a.metrics))

	allowedOrigins := a.cfg.CORSAllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173"}
	}

	corsConfig := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// Rate Limiter
	router.Use(middleware.RateLimiter(
		a.cfg.RateLimitEnabled,
		a.cfg.RateLimitLimit,
		a.cfg.RateLimitBurst,
		"events:global",
		a.recordRateLimitHit,
	))

	a.registerHealthRoutes(router)
	a.registerAPIRoutes(router, cfg)

	return router
}

func (a *Application) registerHealthRoutes(router *gin.Engine) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := a.mongoClient.Ping(ctx, nil); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
}

func (a *Application) registerAPIRoutes(router *gin.Engine, cfg RouterConfig) {
	authMiddleware := middleware.AuthMiddleware(
		a.cfg.JWTSecret,
		a.redisClusterClient(),
		middleware.WithFailClosedResponse(http.StatusServiceUnavailable, "authentication temporarily unavailable"),
	)
	api := router.Group("/api", authMiddleware)

	eventGroup := api.Group("/events")
	{
		eventGroup.POST("", a.eventActionLimiter(middleware.CreateEventRateLimit), cfg.EventController.CreateEvent)
		eventGroup.GET("", cfg.EventController.ListEvents)
		eventGroup.GET("/my-events", cfg.EventController.GetMyEvents)
		eventGroup.GET("/birthdays", cfg.EventController.GetBirthdays)
		eventGroup.GET("/categories", cfg.EventController.GetCategories)
		eventGroup.GET("/recommendations", a.eventActionLimiter(middleware.RecommendationRateLimit), cfg.EventController.GetRecommendations)
		eventGroup.GET("/trending", a.eventActionLimiter(middleware.TrendingRateLimit), cfg.EventController.GetTrending)
		eventGroup.GET("/search", a.eventActionLimiter(middleware.SearchRateLimit), cfg.EventController.SearchEvents)
		eventGroup.GET("/nearby", a.eventActionLimiter(middleware.SearchRateLimit), cfg.EventController.GetNearbyEvents)
		eventGroup.GET("/invitations", cfg.EventController.GetInvitations)
		eventGroup.POST("/invitations/:id/respond", cfg.EventController.RespondToInvitation)
		eventGroup.GET("/:id", cfg.EventController.GetEvent)
		eventGroup.PUT("/:id", cfg.EventController.UpdateEvent)
		eventGroup.DELETE("/:id", cfg.EventController.DeleteEvent)
		eventGroup.POST("/:id/rsvp", a.eventActionLimiter(middleware.RSVPRateLimit), cfg.EventController.RSVP)
		eventGroup.POST("/:id/share", cfg.EventController.ShareEvent)
		eventGroup.POST("/:id/invite", a.eventActionLimiter(middleware.InviteRateLimit), cfg.EventController.InviteFriends)
		eventGroup.GET("/:id/attendees", cfg.EventController.GetAttendees)
		eventGroup.POST("/:id/co-hosts", cfg.EventController.AddCoHost)
		eventGroup.DELETE("/:id/co-hosts/:userId", cfg.EventController.RemoveCoHost)
		eventGroup.POST("/:id/posts", a.eventActionLimiter(middleware.EventPostRateLimit), cfg.EventController.CreatePost)
		eventGroup.GET("/:id/posts", cfg.EventController.GetPosts)
		eventGroup.DELETE("/:id/posts/:postId", cfg.EventController.DeletePost)
		eventGroup.POST("/:id/posts/:postId/react", cfg.EventController.ReactToPost)
	}
}

func (a *Application) eventActionLimiter(config middleware.EventRateLimitConfig) gin.HandlerFunc {
	if a.redisClient == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return middleware.EventRateLimiter(a.redisClient, config, a.recordRateLimitHit)
}

func (a *Application) redisClusterClient() *goredis.ClusterClient {
	if a.redisClient == nil {
		return nil
	}
	return a.redisClient.ClusterClient
}

func (a *Application) recordRateLimitHit(action string) {
	if a.businessMetrics == nil || action == "" {
		return
	}
	a.businessMetrics.RecordRateLimitHit(action)
}
