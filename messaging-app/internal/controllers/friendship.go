package controllers

import (
	"log"
	"messaging-app/internal/friendshipclient"
	"net/http"
	"strconv"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FriendshipController struct {
	client *friendshipclient.Client
}

func NewFriendshipController(client *friendshipclient.Client) *FriendshipController {
	return &FriendshipController{client}
}

type friendshipRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
}

type friendshipResponse struct {
	FriendshipID string `json:"friendship_id" binding:"required"`
	Accept       bool   `json:"accept"`
}

// @Summary Send friend request
// @Description Send a friend request to another user
// @Tags friendships
// @Accept json
// @Produce json
// @Param request body friendshipRequest true "Friend request details"
// @Success 201 {object} models.Friendship
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /friendships/requests [post]
func (c *FriendshipController) SendRequest(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	requesterID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req friendshipRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receiverID, err := primitive.ObjectIDFromHex(req.ReceiverID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid receiver ID"})
		return
	}

	friendship, err := c.client.SendRequest(ctx.Request.Context(), requesterID, receiverID)
	if err != nil {
		st, ok := status.FromError(err)
		code := http.StatusBadRequest
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				code = http.StatusConflict
			case codes.InvalidArgument:
				code = http.StatusBadRequest
			default:
				code = http.StatusInternalServerError
			}
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, friendship)
}

// @Summary Respond to friend request
// @Description Accept or reject a friend request
// @Tags friendships
// @Accept json
// @Produce json
// @Param request body friendshipResponse true "Response details"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /friendships/requests/respond [post]
func (c *FriendshipController) RespondToRequest(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	receiverID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req friendshipResponse
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	friendshipID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		log.Printf("Error parsing friendship ID from path: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid friendship ID"})
		return
	}

	if err := c.client.RespondToRequest(ctx.Request.Context(), friendshipID, receiverID, req.Accept); err != nil {
		st, ok := status.FromError(err)
		code := http.StatusBadRequest
		if ok {
			switch st.Code() {
			case codes.NotFound:
				code = http.StatusNotFound
			case codes.PermissionDenied:
				code = http.StatusForbidden
			default:
				code = http.StatusInternalServerError
			}
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// @Summary List friendships
// @Description Get a list of friendships with optional status filter
// @Tags friendships
// @Produce json
// @Param status query string false "Friendship status (pending/accepted/rejected)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /friendships [get]
func (c *FriendshipController) ListFriendships(ctx *gin.Context) {
	uid, exists := ctx.Get("userID")
	if !exists {
		log.Println("userID not set in context")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	userID := uid.(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	status := ctx.Query("status")

	page, err := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)
	if err != nil || limit < 1 {
		limit = 10
	}

	friendships, total, err := c.client.ListFriendships(ctx.Request.Context(), currentUserID, models.FriendshipStatus(status), page, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":       friendships,
		"total":      total,
		"page":       page,
		"totalPages": (total + limit - 1) / limit,
	})
}

// @Summary Search friends
// @Description Search for friends by username or full name
// @Tags friendships
// @Produce json
// @Param query query string true "Search query"
// @Param limit query int false "Max results" default(20)
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /friendships/search [get]
func (c *FriendshipController) SearchFriends(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	query := ctx.Query("query")
	if query == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	limit, err := strconv.ParseInt(ctx.DefaultQuery("limit", "20"), 10, 64)
	if err != nil || limit < 1 {
		limit = 20
	}

	friends, err := c.client.SearchFriends(ctx.Request.Context(), currentUserID, query, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, friends)
}

// @Summary Get detailed friendship status
// @Description Get detailed friendship status between the current user and another user
// @Tags friendships
// @Produce json
// @Param other_user_id query string true "Other user ID to check"
// @Success 200 {object} services.FriendshipStatusResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /friendships/check [get]
func (c *FriendshipController) CheckFriendship(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	otherUserID, err := primitive.ObjectIDFromHex(ctx.Query("other_user_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid other user ID"})
		return
	}

	status, err := c.client.GetDetailedFriendshipStatus(ctx.Request.Context(), currentUserID, otherUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, status)
}

// @Summary Unfriend a user
// @Description Remove a friendship between two users
// @Tags friendships
// @Produce json
// @Param friend_id path string true "Friend ID to unfriend"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /friendships/{friend_id} [delete]
func (c *FriendshipController) Unfriend(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	friendID, err := primitive.ObjectIDFromHex(ctx.Param("friend_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid friend ID"})
		return
	}

	if err := c.client.Unfriend(ctx.Request.Context(), currentUserID, friendID); err != nil {
		st, ok := status.FromError(err)
		code := http.StatusBadRequest
		if ok && st.Code() == codes.NotFound {
			code = http.StatusNotFound
		} else if ok {
			code = http.StatusInternalServerError
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// @Summary Block a user
// @Description Block another user
// @Tags friendships
// @Accept json
// @Produce json
// @Param user_id path string true "User ID to block"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 409 {object} gin.H
// @Router /friendships/block/{user_id} [post]
func (c *FriendshipController) BlockUser(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	blockerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	blockedID, err := primitive.ObjectIDFromHex(ctx.Param("user_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID to block"})
		return
	}

	if err := c.client.BlockUser(ctx.Request.Context(), blockerID, blockedID); err != nil {
		code := http.StatusInternalServerError
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				code = http.StatusConflict
			case codes.InvalidArgument:
				code = http.StatusBadRequest
			}
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// @Summary Unblock a user
// @Description Remove a block between users
// @Tags friendships
// @Produce json
// @Param user_id path string true "User ID to unblock"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /friendships/block/{user_id} [delete]
func (c *FriendshipController) UnblockUser(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	blockerID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	blockedID, err := primitive.ObjectIDFromHex(ctx.Param("user_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID to unblock"})
		return
	}

	if err := c.client.UnblockUser(ctx.Request.Context(), blockerID, blockedID); err != nil {
		code := http.StatusInternalServerError
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.NotFound {
			code = http.StatusNotFound
		}
		ctx.JSON(code, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// @Summary Check if user is blocked
// @Description Check if a user is blocked by another user
// @Tags friendships
// @Produce json
// @Param user_id path string true "User ID to check block status"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /friendships/block/{user_id}/status [get]
func (c *FriendshipController) IsBlocked(ctx *gin.Context) {
	// Not implemented in client explicitly as single method, but we can assume 'CheckFriendship' or specific 'IsBlocked'.
	// Wait, I implemented 'GetDetailedFriendshipStatus' which has block info.
	// Or did I implement IsBlocked in client?
	// Checking client.go... I didn't verify IsBlocked exists in client explicitly, but I likely missed it.
	// Let's implement it in controller via GetBlockedUsers or similar if not available, OR rely on GetDetailedFriendshipStatus.
	// Actually, I should use GetDetailed.
	// Re-reading client logic: I *only* implemented what was in proto. Proto had CheckFriendship.
	// Proto CheckFriendshipResponse has IsBlockedByUser, IsBlockedByTarget.
	// So I can use CheckFriendship to implement IsBlocked.

	userID := ctx.MustGet("userID").(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	otherUserID, err := primitive.ObjectIDFromHex(ctx.Param("user_id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid other user ID"})
		return
	}

	// Use GetDetailedFriendshipStatus to check block status
	detailedStatus, err := c.client.GetDetailedFriendshipStatus(ctx.Request.Context(), currentUserID, otherUserID)
	if err != nil {
		st, _ := status.FromError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"is_blocked_by_viewer": detailedStatus.IsBlockedByViewer,
		"has_blocked_viewer":   detailedStatus.HasBlockedViewer,
	})
}

// @Summary Get blocked users list
// @Description Get a list of users blocked by the current user
// @Tags friendships
// @Produce json
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Router /friendships/blocked [get]
func (c *FriendshipController) GetBlockedUsers(ctx *gin.Context) {
	userID := ctx.MustGet("userID").(string)
	currentUserID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	blockedUsers, err := c.client.GetBlockedUsers(ctx.Request.Context(), currentUserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"blocked_users": blockedUsers})
}
