package platform

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/spydotech-group/events-service/config"
	"gitlab.com/spydotech-group/events-service/internal/controllers"

	"gitlab.com/spydotech-group/shared-entity/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	router.Use(middleware.RateLimiter(a.cfg.RateLimitEnabled, a.cfg.RateLimitLimit, a.cfg.RateLimitBurst))

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
	authMiddleware := middleware.AuthMiddleware(a.cfg.JWTSecret, a.redisClient.GetClient())
	api := router.Group("/api", authMiddleware)

	eventGroup := api.Group("/events")
	{
		eventGroup.POST("", cfg.EventController.CreateEvent)
		eventGroup.GET("", cfg.EventController.ListEvents)
		eventGroup.GET("/my-events", cfg.EventController.GetMyEvents)
		eventGroup.GET("/birthdays", cfg.EventController.GetBirthdays)
		eventGroup.GET("/categories", cfg.EventController.GetCategories)
		eventGroup.GET("/recommendations", cfg.EventController.GetRecommendations)
		eventGroup.GET("/trending", cfg.EventController.GetTrending)
		eventGroup.GET("/search", cfg.EventController.SearchEvents)
		eventGroup.GET("/nearby", cfg.EventController.GetNearbyEvents)
		eventGroup.GET("/invitations", cfg.EventController.GetInvitations)
		eventGroup.POST("/invitations/:id/respond", cfg.EventController.RespondToInvitation)
		eventGroup.GET("/:id", cfg.EventController.GetEvent)
		eventGroup.PUT("/:id", cfg.EventController.UpdateEvent)
		eventGroup.DELETE("/:id", cfg.EventController.DeleteEvent)
		eventGroup.POST("/:id/rsvp", cfg.EventController.RSVP)
		eventGroup.POST("/:id/share", cfg.EventController.ShareEvent)
		eventGroup.POST("/:id/invite", cfg.EventController.InviteFriends)
		eventGroup.GET("/:id/attendees", cfg.EventController.GetAttendees)
		eventGroup.POST("/:id/co-hosts", cfg.EventController.AddCoHost)
		eventGroup.DELETE("/:id/co-hosts/:userId", cfg.EventController.RemoveCoHost)
		eventGroup.POST("/:id/posts", cfg.EventController.CreatePost)
		eventGroup.GET("/:id/posts", cfg.EventController.GetPosts)
		eventGroup.DELETE("/:id/posts/:postId", cfg.EventController.DeletePost)
		eventGroup.POST("/:id/posts/:postId/react", cfg.EventController.ReactToPost)
	}
}
