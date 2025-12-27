package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MuhibNayem/connectify-v2/reel-service/config"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/reel-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	userpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/MuhibNayem/connectify-v2/shared-entity/redis"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type ReelService interface {
	CreateReel(ctx context.Context, userID primitive.ObjectID, author models.PostAuthor, req service.CreateReelRequest) (*models.Reel, error)
	GetReel(ctx context.Context, reelID primitive.ObjectID) (*models.Reel, error)
	GetUserReels(ctx context.Context, userID primitive.ObjectID) ([]models.Reel, error)
	DeleteReel(ctx context.Context, reelID, userID primitive.ObjectID) error
	GetReelsFeed(ctx context.Context, viewerID primitive.ObjectID, limit, offset int64) ([]models.Reel, error)
	IncrementViews(ctx context.Context, reelID, viewerID primitive.ObjectID) error
	ReactToReel(ctx context.Context, reelID, userID primitive.ObjectID, reactionType models.ReactionType) error
	AddComment(ctx context.Context, reelID, userID primitive.ObjectID, content string, author models.PostAuthor, mentions []primitive.ObjectID) (*models.Comment, error)
	GetComments(ctx context.Context, reelID primitive.ObjectID, limit, offset int64) ([]models.Comment, error)
	AddReply(ctx context.Context, reelID, commentID, userID primitive.ObjectID, content string, author models.PostAuthor) (*models.Reply, error)
	ReactToComment(ctx context.Context, reelID, commentID, userID primitive.ObjectID, reactionType models.ReactionType) error
}

type ReelHandler struct {
	reelService ReelService
	userClient  userpb.UserServiceClient
	metrics     *metrics.BusinessMetrics
}

func NewReelHandler(reelService ReelService, userClient userpb.UserServiceClient, metrics *metrics.BusinessMetrics) *ReelHandler {
	return &ReelHandler{
		reelService: reelService,
		userClient:  userClient,
		metrics:     metrics,
	}
}

func BuildRouter(cfg *config.Config, handler *ReelHandler, redisClient *redis.ClusterClient) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("reel-service"))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api/v1")
	{
		reels := api.Group("/reels")
		{
			reels.POST("", handler.CreateReel)
			reels.GET("/feed", handler.GetReelsFeed)
			reels.GET("/:id", handler.GetReel)
			reels.DELETE("/:id", handler.DeleteReel)
			reels.POST("/:id/view", handler.IncrementViews)
			reels.POST("/:id/react", handler.ReactToReel)
			reels.GET("/:id/comments", handler.GetComments)
			reels.POST("/:id/comments", handler.AddComment)
			reels.POST("/:id/comments/:commentId/replies", handler.AddReply)
			reels.POST("/:id/comments/:commentId/react", handler.ReactToComment)
		}

		users := api.Group("/users")
		{
			users.GET("/:id/reels", handler.GetUserReels)
		}
	}

	return router
}

func (h *ReelHandler) CreateReel(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req service.CreateReelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	author, err := h.getAuthor(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	reel, err := h.reelService.CreateReel(c.Request.Context(), objUserID, author, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reel)
}

func (h *ReelHandler) GetReelsFeed(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)

	reels, err := h.reelService.GetReelsFeed(c.Request.Context(), objUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reels)
}

func (h *ReelHandler) GetReel(c *gin.Context) {
	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	reel, err := h.reelService.GetReel(c.Request.Context(), reelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reel not found"})
		return
	}

	c.JSON(http.StatusOK, reel)
}

func (h *ReelHandler) GetUserReels(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	reels, err := h.reelService.GetUserReels(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reels)
}

func (h *ReelHandler) DeleteReel(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	if err := h.reelService.DeleteReel(c.Request.Context(), reelID, objUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reel deleted"})
}

func (h *ReelHandler) IncrementViews(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	if err := h.reelService.IncrementViews(c.Request.Context(), reelID, objUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "View recorded"})
}

func (h *ReelHandler) ReactToReel(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	var req struct {
		Type models.ReactionType `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.reelService.ReactToReel(c.Request.Context(), reelID, objUserID, req.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reacted successfully"})
}

func (h *ReelHandler) GetComments(c *gin.Context) {
	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)

	comments, err := h.reelService.GetComments(c.Request.Context(), reelID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func (h *ReelHandler) AddComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	var req struct {
		Content  string               `json:"content" binding:"required"`
		Mentions []primitive.ObjectID `json:"mentions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	author, err := h.getAuthor(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	comment, err := h.reelService.AddComment(c.Request.Context(), reelID, objUserID, req.Content, author, req.Mentions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

func (h *ReelHandler) AddReply(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	commentID, err := primitive.ObjectIDFromHex(c.Param("commentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	author, err := h.getAuthor(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	reply, err := h.reelService.AddReply(c.Request.Context(), reelID, commentID, objUserID, req.Content, author)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (h *ReelHandler) ReactToComment(c *gin.Context) {
	userID, _ := c.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	reelID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reel ID"})
		return
	}

	commentID, err := primitive.ObjectIDFromHex(c.Param("commentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req struct {
		ReactionType models.ReactionType `json:"reaction_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.reelService.ReactToComment(c.Request.Context(), reelID, commentID, objUserID, req.ReactionType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reacted successfully"})
}

func (h *ReelHandler) getAuthor(ctx context.Context, userID string) (models.PostAuthor, error) {
	resp, err := h.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
	if err != nil {
		return models.PostAuthor{}, err
	}

	return models.PostAuthor{
		ID:       resp.User.Id,
		Username: resp.User.Username,
		Avatar:   resp.User.Avatar,
		FullName: resp.User.FullName,
	}, nil
}
