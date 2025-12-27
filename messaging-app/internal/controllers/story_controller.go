package controllers

import (
	"messaging-app/internal/repositories"
	"messaging-app/internal/storyclient"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StoryController struct {
	storyClient    *storyclient.Client
	friendshipRepo *repositories.FriendshipRepository
}

func NewStoryController(storyClient *storyclient.Client, friendshipRepo *repositories.FriendshipRepository) *StoryController {
	return &StoryController{
		storyClient:    storyClient,
		friendshipRepo: friendshipRepo,
	}
}

func (c *StoryController) CreateStory(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateStoryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	story, err := c.storyClient.CreateStory(ctx.Request.Context(), objUserID, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, story)
}

func (c *StoryController) GetStoriesFeed(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	objUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Limit  int `form:"limit"`
		Offset int `form:"offset"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		req.Limit = 10
		req.Offset = 0
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Get friends for privacy filtering
	friends, err := c.friendshipRepo.GetFriends(ctx.Request.Context(), objUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	friendIDs := make([]primitive.ObjectID, len(friends))
	for i, friend := range friends {
		friendIDs[i] = friend.ID
	}

	stories, err := c.storyClient.GetStoriesFeed(ctx.Request.Context(), objUserID, friendIDs, req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stories)
}

func (c *StoryController) GetUserStories(ctx *gin.Context) {
	targetUserIDStr := ctx.Param("id")
	targetUserID, err := primitive.ObjectIDFromHex(targetUserIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	stories, err := c.storyClient.GetUserStories(ctx.Request.Context(), targetUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, stories)
}

func (c *StoryController) DeleteStory(ctx *gin.Context) {
	storyIDStr := ctx.Param("id")
	storyID, err := primitive.ObjectIDFromHex(storyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid story ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	if err := c.storyClient.DeleteStory(ctx.Request.Context(), storyID, objUserID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *StoryController) ViewStory(ctx *gin.Context) {
	storyIDStr := ctx.Param("id")
	storyID, err := primitive.ObjectIDFromHex(storyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid story ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	if err := c.storyClient.RecordView(ctx.Request.Context(), storyID, objUserID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *StoryController) ReactToStory(ctx *gin.Context) {
	storyIDStr := ctx.Param("id")
	storyID, err := primitive.ObjectIDFromHex(storyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid story ID"})
		return
	}

	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := ctx.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	if err := c.storyClient.ReactToStory(ctx.Request.Context(), storyID, objUserID, req.Type); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *StoryController) GetStoryViewers(ctx *gin.Context) {
	storyIDStr := ctx.Param("id")
	storyID, err := primitive.ObjectIDFromHex(storyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid story ID"})
		return
	}

	userID, _ := ctx.Get("userID")
	objUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	viewers, err := c.storyClient.GetStoryViewers(ctx.Request.Context(), storyID, objUserID)
	if err != nil {
		// unauthorized or not found
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, viewers)
}
