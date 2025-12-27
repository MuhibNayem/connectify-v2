package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/story-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoryHandler struct {
	storyService *service.StoryService
}

func NewStoryHandler(storyService *service.StoryService) *StoryHandler {
	return &StoryHandler{storyService: storyService}
}

func (h *StoryHandler) RegisterRoutes(router *gin.Engine, auth gin.HandlerFunc) {
	api := router.Group("/api")
	stories := api.Group("/stories")
	stories.Use(auth)
	{
		stories.POST("", h.CreateStory)
		stories.GET("/feed", h.GetStoriesFeed)
		stories.GET("/user/:id", h.GetUserStories)
		stories.GET("/:id", h.GetStory)
		stories.DELETE("/:id", h.DeleteStory)
		stories.POST("/:id/view", h.RecordView)
		stories.POST("/:id/react", h.ReactToStory)
		stories.GET("/:id/viewers", h.GetStoryViewers)
	}
}

type createStoryRequest struct {
	MediaURL       string   `json:"media_url" binding:"required"`
	MediaType      string   `json:"media_type" binding:"required"`
	Privacy        string   `json:"privacy"`
	AllowedViewers []string `json:"allowed_viewers"`
	BlockedViewers []string `json:"blocked_viewers"`
}

func (h *StoryHandler) CreateStory(c *gin.Context) {
	var req createStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, err)
		return
	}

	if req.MediaURL == "" || req.MediaType == "" {
		respondWithMessage(c, http.StatusBadRequest, "media_url and media_type are required")
		return
	}

	userID, err := h.userIDFromContext(c)
	if err != nil {
		respondWithError(c, http.StatusUnauthorized, err)
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
		respondWithError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, story)
}

func (h *StoryHandler) GetStory(c *gin.Context) {
	storyID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		respondWithMessage(c, http.StatusBadRequest, "invalid story id")
		return
	}

	story, err := h.storyService.GetStory(c.Request.Context(), storyID)
	if err != nil {
		respondWithError(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, story)
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

	friendIDs := parseObjectIDs(strings.Split(c.DefaultQuery("friend_ids", ""), ","))

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
		respondWithError(c, http.StatusInternalServerError, err)
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
		respondWithError(c, http.StatusInternalServerError, err)
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

func respondWithError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}

func respondWithMessage(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{
		"error": msg,
	})
}

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
