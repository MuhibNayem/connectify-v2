package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MuhibNayem/connectify-v2/shared-entity/middleware"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	userpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/user/v1"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/metrics"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoryHandler struct {
	storyService      StoryService
	userClient        userpb.UserServiceClient
	rateLimitObserver func(action string)
}

func NewStoryHandler(storyService StoryService, userClient userpb.UserServiceClient, businessMetrics *metrics.BusinessMetrics) *StoryHandler {
	var observer func(action string)
	if businessMetrics != nil {
		observer = businessMetrics.RecordRateLimitHit
	}

	return &StoryHandler{
		storyService:      storyService,
		userClient:        userClient,
		rateLimitObserver: observer,
	}
}

func (h *StoryHandler) RegisterRoutes(router *gin.Engine, auth gin.HandlerFunc) {
	api := router.Group("/api")
	stories := api.Group("/stories")
	stories.Use(auth)
	{
		stories.POST("",
			middleware.StrictRateLimiter(0.2, 5, "stories:create", h.rateLimitObserver), // 12 per min
			h.CreateStory,
		)
		stories.GET("/feed",
			middleware.StrictRateLimiter(2, 10, "stories:feed", h.rateLimitObserver), // 120 per min
			h.GetStoriesFeed,
		)
		stories.GET("/user/:id",
			middleware.StrictRateLimiter(1, 8, "stories:user", h.rateLimitObserver), // 60 per min
			h.GetUserStories,
		)
		stories.GET("/:id",
			middleware.StrictRateLimiter(3, 15, "stories:view", h.rateLimitObserver), // 180 per min
			h.GetStory,
		)
		stories.DELETE("/:id",
			middleware.StrictRateLimiter(0.3, 3, "stories:delete", h.rateLimitObserver), // 18 per min
			h.DeleteStory,
		)
		stories.POST("/:id/view",
			middleware.StrictRateLimiter(5, 20, "stories:track_view", h.rateLimitObserver), // 300 per min
			h.RecordView,
		)
		stories.POST("/:id/react",
			middleware.StrictRateLimiter(1, 10, "stories:react", h.rateLimitObserver), // 60 per min
			h.ReactToStory,
		)
		stories.GET("/:id/viewers",
			middleware.StrictRateLimiter(0.5, 5, "stories:viewers", h.rateLimitObserver), // 30 per min
			h.GetStoryViewers,
		)
	}
}

type createStoryRequest struct {
	MediaURL       string   `json:"media_url"`
	MediaType      string   `json:"media_type"`
	Privacy        string   `json:"privacy"`
	AllowedViewers []string `json:"allowed_viewers"`
	BlockedViewers []string `json:"blocked_viewers"`
}

func (h *StoryHandler) CreateStory(c *gin.Context) {
	var req createStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request format", ErrCodeValidation)
		return
	}

	if req.MediaURL == "" || req.MediaType == "" {
		RespondWithValidationError(c, "Missing required fields", map[string]string{
			"media_url":  "Media URL is required",
			"media_type": "Media type is required",
		})
		return
	}

	userID, err := h.userIDFromContext(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	author := models.PostAuthor{
		ID: userID.Hex(),
	}

	serviceReq := service.CreateStoryRequest{
		MediaURL:       req.MediaURL,
		MediaType:      req.MediaType,
		Privacy:        models.PrivacySettingType(req.Privacy),
		AllowedViewers: req.AllowedViewers,
		BlockedViewers: req.BlockedViewers,
	}

	story, err := h.storyService.CreateStory(c.Request.Context(), userID, author, serviceReq)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			RespondWithError(c, http.StatusBadRequest, err.Error(), ErrCodeValidation)
		} else {
			RespondWithError(c, http.StatusInternalServerError, "Failed to create story", ErrCodeInternalError)
		}
		return
	}

	RespondWithSuccess(c, http.StatusCreated, "Story created successfully", story)
}

func (h *StoryHandler) GetStory(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid story ID format", ErrCodeValidation)
		return
	}

	viewerID, err := h.userIDFromContext(c)
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "Authentication required", ErrCodeUnauthorized)
		return
	}

	story, err := h.storyService.GetStory(c.Request.Context(), storyID, viewerID)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "Story not found or access denied", ErrCodeStoryNotFound)
		return
	}

	RespondWithData(c, http.StatusOK, story)
}

func (h *StoryHandler) DeleteStory(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid story id")
		return
	}

	userID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
		return
	}

	if err := h.storyService.DeleteStory(c.Request.Context(), storyID, userID); err != nil {
		respondWithError(c, http.StatusForbidden, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *StoryHandler) GetStoriesFeed(c *gin.Context) {
	userID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	friendIDs := h.fetchFriendIDs(c.Request.Context(), userID)
	if query := strings.TrimSpace(c.DefaultQuery("friend_ids", "")); query != "" {
		friendIDs = append(friendIDs, parseObjectIDs(strings.Split(query, ","))...)
	}

	stories, err := h.storyService.GetStoriesFeed(c.Request.Context(), userID, friendIDs, limit, offset)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stories": stories,
		"count":   len(stories),
	})
}

func (h *StoryHandler) GetUserStories(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := primitive.ObjectIDFromHex(userIDParam)
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid user id")
		return
	}

	stories, err := h.storyService.GetUserStories(c.Request.Context(), userID)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stories": stories,
	})
}

func (h *StoryHandler) RecordView(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid story id")
		return
	}

	viewerID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
		return
	}

	if err := h.storyService.RecordView(c.Request.Context(), storyID, viewerID); err != nil {
		if err.Error() == "story not found" {
			respondWithError(c, http.StatusNotFound, err)
		} else {
			respondWithError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.Status(http.StatusNoContent)
}

type reactRequest struct {
	ReactionType string `json:"reaction_type" binding:"required"`
}

func (h *StoryHandler) ReactToStory(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid story id")
		return
	}

	userID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
		return
	}

	var req reactRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ReactionType == "" {
		respondWithMessage(c, http.StatusBadRequest, "reaction_type is required")
		return
	}

	if err := h.storyService.ReactToStory(c.Request.Context(), storyID, userID, req.ReactionType); err != nil {
		if err.Error() == "story not found" {
			respondWithError(c, http.StatusNotFound, err)
		} else if err.Error() == "invalid reaction type" {
			respondWithError(c, http.StatusBadRequest, err)
		} else {
			respondWithError(c, http.StatusInternalServerError, err)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reaction recorded"})
}

func (h *StoryHandler) GetStoryViewers(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid story id")
		return
	}

	userID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
		return
	}

	viewers, err := h.storyService.GetStoryViewers(c.Request.Context(), storyID, userID)
	if err != nil {
		respondWithError(c, http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"viewers": viewers,
	})
}

func (h *StoryHandler) userIDFromContext(c *gin.Context) (primitive.ObjectID, error) {
	raw, exists := c.Get("user_id")
	if !exists {
		return primitive.NilObjectID, errUnauthorized
	}

	switch v := raw.(type) {
	case string:
		return primitive.ObjectIDFromHex(v)
	case primitive.ObjectID:
		return v, nil
	default:
		return primitive.NilObjectID, errUnauthorized
	}
}

var errUnauthorized = errors.New("authentication required")

func parseObjectIDs(values []string) []primitive.ObjectID {
	results := make([]primitive.ObjectID, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if id, err := primitive.ObjectIDFromHex(v); err == nil {
			results = append(results, id)
		}
	}
	return results
}

func (h *StoryHandler) fetchFriendIDs(ctx context.Context, userID primitive.ObjectID) []primitive.ObjectID {
	if h.userClient == nil {
		return []primitive.ObjectID{}
	}

	resp, err := h.userClient.GetFriendIDs(ctx, &userpb.GetFriendIDsRequest{UserId: userID.Hex()})
	if err != nil || resp == nil || len(resp.FriendIds) == 0 {
		return []primitive.ObjectID{}
	}

	return parseObjectIDs(resp.FriendIds)
}
