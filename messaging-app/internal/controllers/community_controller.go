package controllers

import (
	"net/http"
	"strconv"

	"sync"
	"time"

	"messaging-app/internal/communityclient"
	"messaging-app/internal/storageclient"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/MuhibNayem/connectify-v2/shared-entity/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CommunityController struct {
	communityClient *communityclient.Client
	storageClient   *storageclient.Client
}

func NewCommunityController(communityClient *communityclient.Client, storageClient *storageclient.Client) *CommunityController {
	return &CommunityController{
		communityClient: communityClient,
		storageClient:   storageClient,
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

	community, err := c.communityClient.CreateCommunity(ctx.Request.Context(), userID, req)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signCommunityMedia(ctx, community)

	ctx.JSON(http.StatusCreated, community)
}

func (c *CommunityController) GetCommunity(ctx *gin.Context) {
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	var viewerID primitive.ObjectID
	if userID, err := utils.GetUserIDFromContext(ctx); err == nil {
		viewerID = userID
	}

	response, err := c.communityClient.GetCommunity(ctx.Request.Context(), communityID, viewerID)
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
	var viewerID primitive.ObjectID
	if userID, err := utils.GetUserIDFromContext(ctx); err == nil {
		viewerID = userID
	}

	communities, total, err := c.communityClient.ListCommunities(ctx.Request.Context(), viewerID, limit, page, query)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	communityPtrs := make([]*models.CommunityResponse, len(communities))
	for i := range communities {
		communityPtrs[i] = &communities[i]
	}
	c.signCommunityResponse(ctx, communityPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"communities": communityPtrs,
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
	targetID := userID

	targetUserIDParam := ctx.Param("userId")
	if targetUserIDParam != "" {
		if tid, err := primitive.ObjectIDFromHex(targetUserIDParam); err == nil {
			targetID = tid
		}
	}

	communities, err := c.communityClient.GetUserCommunities(ctx.Request.Context(), targetID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	communityPtrs := make([]*models.Community, len(communities))
	for i := range communities {
		communityPtrs[i] = &communities[i]
	}

	c.signCommunityMedia(ctx, communityPtrs...)

	ctx.JSON(http.StatusOK, communityPtrs)
}

func (c *CommunityController) JoinCommunity(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	if err := c.communityClient.JoinCommunity(ctx.Request.Context(), communityID, userID); err != nil {
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
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	if err := c.communityClient.LeaveCommunity(ctx.Request.Context(), communityID, userID); err != nil {
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
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
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
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid target user ID")
		return
	}

	if err := c.communityClient.ApproveMember(ctx.Request.Context(), communityID, actorID, targetID); err != nil {
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
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
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
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid target user ID")
		return
	}

	if err := c.communityClient.RejectMember(ctx.Request.Context(), communityID, actorID, targetID); err != nil {
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
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	var req models.UpdateCommunityRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := c.communityClient.UpdateCommunity(ctx.Request.Context(), communityID, userID, req); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *CommunityController) ListMembers(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)

	var viewerID primitive.ObjectID
	if userID, err := utils.GetUserIDFromContext(ctx); err == nil {
		viewerID = userID
	}

	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	users, total, err := c.communityClient.GetMembers(ctx.Request.Context(), communityID, viewerID, limit, page)
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
	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	users, err := c.communityClient.GetAdmins(ctx.Request.Context(), communityID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, users)
}

func (c *CommunityController) GetPendingMembers(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	communityIDParam := ctx.Param("id")
	communityID, err := primitive.ObjectIDFromHex(communityIDParam)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid community ID")
		return
	}

	users, total, err := c.communityClient.GetPendingMembers(ctx.Request.Context(), communityID, userID, limit, page)
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
