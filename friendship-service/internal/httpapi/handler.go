package httpapi

import (
	"net/http"
	"strconv"

	"github.com/MuhibNayem/connectify-v2/friendship-service/config"
	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/service"
	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	pkgredis "github.com/MuhibNayem/connectify-v2/shared-entity/redis"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type FriendshipHandler struct {
	friendshipService *service.FriendshipService
}

func NewFriendshipHandler(friendshipService *service.FriendshipService) *FriendshipHandler {
	return &FriendshipHandler{
		friendshipService: friendshipService,
	}
}

func BuildRouter(cfg *config.Config, handler *FriendshipHandler, redisClient *pkgredis.ClusterClient) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("friendship-service"))

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

	// Protected API routes
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret, nil))
	{
		friendships := api.Group("/friendships")
		{
			friendships.POST("/requests", handler.SendRequest)
			friendships.POST("/requests/:id/respond", handler.RespondToRequest)
			friendships.GET("", handler.ListFriendships)
			friendships.GET("/search", handler.SearchFriends)
			friendships.GET("/check", handler.CheckFriendship)
			friendships.GET("/status", handler.GetDetailedStatus)
			friendships.DELETE("/:friend_id", handler.Unfriend)
			friendships.POST("/block/:user_id", handler.BlockUser)
			friendships.DELETE("/block/:user_id", handler.UnblockUser)
			friendships.GET("/block/:user_id/status", handler.IsBlocked)
			friendships.GET("/blocked", handler.GetBlockedUsers)
		}
	}

	return router
}

type sendRequestBody struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
}

func (h *FriendshipHandler) SendRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	requesterID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req sendRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver ID"})
		return
	}

	friendship, err := h.friendshipService.SendRequest(c.Request.Context(), requesterID, receiverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, friendship)
}

type respondRequestBody struct {
	Accept bool `json:"accept"`
}

func (h *FriendshipHandler) RespondToRequest(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	receiverID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	friendshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid friendship ID"})
		return
	}

	var req respondRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.friendshipService.RespondToRequest(c.Request.Context(), friendshipID, receiverID, req.Accept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *FriendshipHandler) ListFriendships(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	friendships, total, err := h.friendshipService.ListFriendships(c.Request.Context(), currentUserID, models.FriendshipStatus(status), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       friendships,
		"total":      total,
		"page":       page,
		"totalPages": (total + limit - 1) / limit,
	})
}

func (h *FriendshipHandler) SearchFriends(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)

	friends, err := h.friendshipService.SearchFriends(c.Request.Context(), currentUserID, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, friends)
}

func (h *FriendshipHandler) CheckFriendship(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	otherUserID, err := primitive.ObjectIDFromHex(c.Query("other_user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid other user ID"})
		return
	}

	areFriends, err := h.friendshipService.CheckFriendship(c.Request.Context(), currentUserID, otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"are_friends": areFriends})
}

func (h *FriendshipHandler) GetDetailedStatus(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	otherUserID, err := primitive.ObjectIDFromHex(c.Query("other_user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid other user ID"})
		return
	}

	status, err := h.friendshipService.GetDetailedFriendshipStatus(c.Request.Context(), currentUserID, otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *FriendshipHandler) Unfriend(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	friendID, err := primitive.ObjectIDFromHex(c.Param("friend_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid friend ID"})
		return
	}

	if err := h.friendshipService.Unfriend(c.Request.Context(), currentUserID, friendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *FriendshipHandler) BlockUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	blockerID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	blockedID, err := primitive.ObjectIDFromHex(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID to block"})
		return
	}

	if err := h.friendshipService.BlockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *FriendshipHandler) UnblockUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	blockerID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	blockedID, err := primitive.ObjectIDFromHex(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID to unblock"})
		return
	}

	if err := h.friendshipService.UnblockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *FriendshipHandler) IsBlocked(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	otherUserID, err := primitive.ObjectIDFromHex(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid other user ID"})
		return
	}

	// Use GetDetailedFriendshipStatus to check block status
	status, err := h.friendshipService.GetDetailedFriendshipStatus(c.Request.Context(), currentUserID, otherUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_blocked_by_viewer": status.IsBlockedByViewer,
		"has_blocked_viewer":   status.HasBlockedViewer,
	})
}

func (h *FriendshipHandler) GetBlockedUsers(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	currentUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	blockedUsers, err := h.friendshipService.GetBlockedUsers(c.Request.Context(), currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blocked_users": blockedUsers})
}
