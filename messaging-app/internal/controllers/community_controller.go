package controllers

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"messaging-app/internal/services"
	"messaging-app/internal/storageclient"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/shared-entity/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CommunityController struct {
	communityService *services.CommunityService
	storageClient    *storageclient.Client
}

func NewCommunityController(communityService *services.CommunityService, storageClient *storageclient.Client) *CommunityController {
	return &CommunityController{
		communityService: communityService,
		storageClient:    storageClient,
	}
}

func (c *CommunityController) signCommunityMedia(ctx *gin.Context, communities ...*models.Community) {
	if len(communities) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, com := range communities {
		if com == nil {
			continue
		}
		wg.Add(1)
		go func(cm *models.Community) {
			defer wg.Done()
			var localWg sync.WaitGroup
			if cm.Avatar != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), cm.Avatar, 15*time.Minute)
					if err == nil {
						cm.Avatar = signed
					}
				}()
			}
			if cm.CoverImage != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), cm.CoverImage, 15*time.Minute)
					if err == nil {
						cm.CoverImage = signed
					}
				}()
			}
			localWg.Wait()
		}(com)
	}
	wg.Wait()
}

func (c *CommunityController) signCommunityResponse(ctx *gin.Context, responses ...*models.CommunityResponse) {
	if len(responses) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, res := range responses {
		if res == nil {
			continue
		}
		wg.Add(1)
		go func(r *models.CommunityResponse) {
			defer wg.Done()
			var localWg sync.WaitGroup
			if r.Avatar != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), r.Avatar, 15*time.Minute)
					if err == nil {
						r.Avatar = signed
					}
				}()
			}
			if r.CoverImage != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx.Request.Context(), r.CoverImage, 15*time.Minute)
					if err == nil {
						r.CoverImage = signed
					}
				}()
			}
			localWg.Wait()
		}(res)
	}
	wg.Wait()
}

func (c *CommunityController) CreateCommunity(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req models.CreateCommunityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	community, err := c.communityService.CreateCommunity(ctx, userID, req)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signCommunityMedia(ctx, community)

	ctx.JSON(http.StatusCreated, community)
}

func (c *CommunityController) GetCommunity(ctx *gin.Context) {
	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	userID, _ := utils.GetUserIDFromContext(ctx) // Optional user ID for detailed response

	response, err := c.communityService.GetDetailedCommunityResponse(ctx, communityID, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signCommunityResponse(ctx, response)

	ctx.JSON(http.StatusOK, response)
}

func (c *CommunityController) ListCommunities(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)

	query := ctx.Query("q")

	// Get current user ID to check membership status
	var userID primitive.ObjectID
	currentUserID, err := utils.GetUserIDFromContext(ctx)
	if err == nil {
		userID = currentUserID
	}

	communities, total, err := c.communityService.ListCommunities(ctx, userID, limit, page, query)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs for list
	communityPtrs := make([]*models.CommunityResponse, len(communities))
	for i := range communities {
		communityPtrs[i] = &communities[i]
	}
	c.signCommunityResponse(ctx, communityPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"communities": communities,
		"total":       total,
		"page":        page,
		"limit":       limit,
	})
}

func (c *CommunityController) GetUserCommunities(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Optional: if targeting another user
	targetUserIDParam := ctx.Param("userId")
	if targetUserIDParam != "" {
		targetID, err := primitive.ObjectIDFromHex(targetUserIDParam)
		if err == nil {
			userID = targetID
		}
	}

	communities, err := c.communityService.GetUserCommunities(ctx, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs for list
	// GetUserCommunities returns []models.Community
	communityPtrs := make([]*models.Community, len(communities))
	for i := range communities {
		communityPtrs[i] = &communities[i]
	}
	c.signCommunityMedia(ctx, communityPtrs...)

	ctx.JSON(http.StatusOK, communities)
}

func (c *CommunityController) JoinCommunity(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	if err := c.communityService.JoinCommunity(ctx, communityID, userID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) LeaveCommunity(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	if err := c.communityService.LeaveCommunity(ctx, communityID, userID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) ApproveMember(ctx *gin.Context) {
	actorID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	type Request struct {
		UserID string `json:"user_id" binding:"required"`
	}
	var req Request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	targetID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := c.communityService.ApproveMember(ctx, communityID, actorID, targetID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) RejectMember(ctx *gin.Context) {
	actorID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	type Request struct {
		UserID string `json:"user_id" binding:"required"`
	}
	var req Request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	targetID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := c.communityService.RejectMember(ctx, communityID, actorID, targetID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) UpdateSettings(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	var req models.UpdateCommunityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.communityService.UpdateSettings(ctx, communityID, userID, req); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) ListMembers(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)
	viewerID, _ := utils.GetUserIDFromContext(ctx)

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	users, total, err := c.communityService.GetMembers(ctx, communityID, viewerID, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (c *CommunityController) GetAdmins(ctx *gin.Context) {
	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	admins, err := c.communityService.GetAdmins(ctx, communityID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, admins)
}

func (c *CommunityController) GetPendingMembers(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	users, total, err := c.communityService.GetPendingMembers(ctx, communityID, userID, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
