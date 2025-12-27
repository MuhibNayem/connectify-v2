package controllers

import (
	"messaging-app/internal/reelclient"
	"messaging-app/internal/storageclient"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	reelpb "github.com/MuhibNayem/connectify-v2/shared-entity/proto/reel/v1"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReelController struct {
	reelClient    *reelclient.Client
	storageClient *storageclient.Client
}

func NewReelController(reelClient *reelclient.Client, storageClient *storageclient.Client) *ReelController {
	return &ReelController{
		reelClient:    reelClient,
		storageClient: storageClient,
	}
}

func (c *ReelController) CreateReel(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateReelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	objUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Map local model req to proto
	// Warning: models.CreateReelRequest might differ from proto request fields
	// Let's assume we copy fields manually
	protoReq := &reelpb.CreateReelRequest{
		UserId:       objUserID.Hex(),
		VideoUrl:     req.VideoURL,
		ThumbnailUrl: req.ThumbnailURL,
		Caption:      req.Caption,
		Duration:     float64(req.Duration),
		Privacy:      string(req.Privacy),
	}
	// Convert ObjectIDs to strings for viewers
	for _, id := range req.AllowedViewers {
		protoReq.AllowedViewers = append(protoReq.AllowedViewers, id.Hex())
	}
	for _, id := range req.BlockedViewers {
		protoReq.BlockedViewers = append(protoReq.BlockedViewers, id.Hex())
	}

	reel, err := c.reelClient.CreateReel(ctx.Request.Context(), protoReq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, reel)
}

// Helper to sign a single reel's URLs
func (c *ReelController) signReelURLs(ctx *gin.Context, reel *reelpb.Reel) {
	if reel == nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if reel.VideoUrl != "" {
			signedURL, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), reel.VideoUrl, 15*time.Minute)
			if err == nil {
				reel.VideoUrl = signedURL
			}
		}
	}()

	go func() {
		defer wg.Done()
		if reel.ThumbnailUrl != "" {
			signedURL, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), reel.ThumbnailUrl, 15*time.Minute)
			if err == nil {
				reel.ThumbnailUrl = signedURL
			}
		}
	}()

	wg.Wait()
}

func (c *ReelController) GetReelsFeed(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	userID, exists := ctx.Get("userID")
	viewerID := ""
	if exists {
		viewerID = userID.(string)
	}

	reels, err := c.reelClient.GetReelsFeed(ctx.Request.Context(), viewerID, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Concurrent signing for feed
	// We handle each reel in a separate goroutine if list is large?
	// For 10-20 items, a simple loop with internal concurrency (signReelURLs) is fine.
	// But let's go FAANG-scale: fully parallel list processing.
	var wg sync.WaitGroup
	for i := range reels {
		wg.Add(1)
		go func(r *reelpb.Reel) {
			defer wg.Done()
			c.signReelURLs(ctx, r)
		}(reels[i])
	}
	wg.Wait()

	ctx.JSON(http.StatusOK, reels)
}

func (c *ReelController) GetUserReels(ctx *gin.Context) {
	targetUserIDStr := ctx.Param("id")

	reels, err := c.reelClient.GetUserReels(ctx.Request.Context(), targetUserIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var wg sync.WaitGroup
	for i := range reels {
		wg.Add(1)
		go func(r *reelpb.Reel) {
			defer wg.Done()
			c.signReelURLs(ctx, r)
		}(reels[i])
	}
	wg.Wait()

	ctx.JSON(http.StatusOK, reels)
}

func (c *ReelController) GetReel(ctx *gin.Context) {
	reelIDStr := ctx.Param("id")

	reel, err := c.reelClient.GetReel(ctx.Request.Context(), reelIDStr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.signReelURLs(ctx, reel)

	ctx.JSON(http.StatusOK, reel)
}

func (c *ReelController) AddComment(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reelIDStr := ctx.Param("id")

	var req struct {
		Content  string               `json:"content" binding:"required"`
		Mentions []primitive.ObjectID `json:"mentions"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Helper to convert ObjectIDs to strings
	mentions := make([]string, len(req.Mentions))
	for i, m := range req.Mentions {
		mentions[i] = m.Hex()
	}

	comment, err := c.reelClient.AddComment(ctx.Request.Context(), &reelpb.AddCommentRequest{
		ReelId:           reelIDStr,
		UserId:           userID.(string),
		Content:          req.Content,
		ExplicitMentions: mentions,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, comment)
}

func (c *ReelController) GetComments(ctx *gin.Context) {
	reelIDStr := ctx.Param("id")

	limitStr := ctx.DefaultQuery("limit", "20")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	comments, err := c.reelClient.GetComments(ctx.Request.Context(), reelIDStr, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, comments)
}

func (c *ReelController) AddReply(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reelIDStr := ctx.Param("id")
	commentIDStr := ctx.Param("commentId")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reply, err := c.reelClient.AddReply(ctx.Request.Context(), &reelpb.AddReplyRequest{
		ReelId:    reelIDStr,
		CommentId: commentIDStr,
		UserId:    userID.(string),
		Content:   req.Content,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, reply)
}

func (c *ReelController) ReactToComment(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reelIDStr := ctx.Param("id")
	commentIDStr := ctx.Param("commentId")

	var req struct {
		ReactionType models.ReactionType `json:"reaction_type" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.reelClient.ReactToComment(ctx.Request.Context(), reelIDStr, commentIDStr, userID.(string), string(req.ReactionType))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reacted successfully"})
}

func (c *ReelController) IncrementView(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	reelIDStr := ctx.Param("id")

	err := c.reelClient.IncrementView(ctx.Request.Context(), reelIDStr, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "View incremented"})
}

// ReactToReel handles reacting to a reel
func (c *ReelController) ReactToReel(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reelIDStr := ctx.Param("id")

	var req struct {
		Type models.ReactionType `json:"type" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.reelClient.ReactToReel(ctx.Request.Context(), reelIDStr, userID.(string), string(req.Type))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reacted successfully"})
}
