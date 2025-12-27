package controllers

import (
	"context"
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

type EventController struct {
	eventService          services.EventServiceContract
	recommendationService services.EventRecommendationServiceContract
	storageClient         *storageclient.Client
}

func NewEventController(eventService services.EventServiceContract, recommendationService services.EventRecommendationServiceContract, storageClient *storageclient.Client) *EventController {
	return &EventController{
		eventService:          eventService,
		recommendationService: recommendationService,
		storageClient:         storageClient,
	}
}

// GetRecommendations returns personalized event recommendations for the user
func (c *EventController) GetRecommendations(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	limitStr := ctx.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	recommendations, err := c.recommendationService.GetRecommendations(ctx.Request.Context(), userID, limit)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Sign URLs
	if len(recommendations) > 0 {
		eventPtrs := make([]*models.EventResponse, len(recommendations))
		for i := range recommendations {
			eventPtrs[i] = &recommendations[i].Event
		}
		c.signEventResponse(ctx.Request.Context(), eventPtrs...)
	}

	utils.RespondWithSuccess(ctx, recommendations)
}

// GetTrending returns trending events
func (c *EventController) GetTrending(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	trending, err := c.recommendationService.GetTrendingEvents(ctx.Request.Context(), limit)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Sign URLs
	if len(trending) > 0 {
		eventPtrs := make([]*models.Event, 0, len(trending))
		for i := range trending {
			if trending[i].Event != nil {
				eventPtrs = append(eventPtrs, trending[i].Event)
			}
		}
		c.signEvent(ctx.Request.Context(), eventPtrs...)
	}

	utils.RespondWithSuccess(ctx, trending)
}

func (c *EventController) CreateEvent(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req models.CreateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	event, err := c.eventService.CreateEvent(ctx, userID, req)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signEvent(ctx.Request.Context(), event)

	ctx.JSON(http.StatusCreated, event)
}

func (c *EventController) GetEvent(ctx *gin.Context) {
	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	userID, _ := utils.GetUserIDFromContext(ctx) // Optional

	response, err := c.eventService.GetEvent(ctx, eventID, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signEventResponse(ctx.Request.Context(), response)

	ctx.JSON(http.StatusOK, response)
}

func (c *EventController) UpdateEvent(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.UpdateEventRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	response, err := c.eventService.UpdateEvent(ctx, eventID, userID, req)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signEventResponse(ctx.Request.Context(), response)

	ctx.JSON(http.StatusOK, response)
}

func (c *EventController) DeleteEvent(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	if err := c.eventService.DeleteEvent(ctx, eventID, userID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *EventController) ListEvents(ctx *gin.Context) {
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)
	query := ctx.Query("q")
	category := ctx.Query("category")
	period := ctx.Query("period") // today, week, past

	userID, _ := utils.GetUserIDFromContext(ctx)

	events, total, err := c.eventService.ListEvents(ctx, userID, limit, page, query, category, period)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	eventPtrs := make([]*models.EventResponse, len(events))
	for i := range events {
		eventPtrs[i] = &events[i]
	}
	c.signEventResponse(ctx.Request.Context(), eventPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (c *EventController) GetMyEvents(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)

	events, err := c.eventService.GetUserEvents(ctx, userID, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	eventPtrs := make([]*models.EventResponse, len(events))
	for i := range events {
		eventPtrs[i] = &events[i]
	}
	c.signEventResponse(ctx.Request.Context(), eventPtrs...)

	ctx.JSON(http.StatusOK, events)
}

func (c *EventController) RSVP(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.RSVPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.eventService.RSVP(ctx, eventID, userID, req.Status); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *EventController) GetBirthdays(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	response, err := c.eventService.GetFriendBirthdays(ctx, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signBirthdayResponse(ctx.Request.Context(), response)

	ctx.JSON(http.StatusOK, response)
}

// ================================
// Invitation Endpoints
// ================================

// InviteFriends invites friends to an event
func (c *EventController) InviteFriends(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.InviteFriendsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.eventService.InviteFriends(ctx, eventID, userID, req.FriendIDs, req.Message); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// GetInvitations returns pending invitations for the current user
func (c *EventController) GetInvitations(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "10"), 10, 64)

	invitations, total, err := c.eventService.GetUserInvitations(ctx, userID, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	invPtrs := make([]*models.EventInvitationResponse, len(invitations))
	for i := range invitations {
		invPtrs[i] = &invitations[i]
	}
	c.signInvitationResponse(ctx.Request.Context(), invPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"invitations": invitations,
		"total":       total,
		"page":        page,
		"limit":       limit,
	})
}

// RespondToInvitation accepts or declines an invitation
func (c *EventController) RespondToInvitation(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	invitationID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid invitation ID")
		return
	}

	var req models.InvitationRespondRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.eventService.RespondToInvitation(ctx, invitationID, userID, req.Accept); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ================================
// Discussion/Post Endpoints
// ================================

// CreatePost creates a discussion post on an event
func (c *EventController) CreatePost(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.CreateEventPostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	post, err := c.eventService.CreatePost(ctx, eventID, userID, req)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signEventPostResponse(ctx.Request.Context(), post)

	ctx.JSON(http.StatusCreated, post)
}

// GetPosts returns discussion posts for an event
func (c *EventController) GetPosts(ctx *gin.Context) {
	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "20"), 10, 64)

	posts, total, err := c.eventService.GetPosts(ctx, eventID, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	postPtrs := make([]*models.EventPostResponse, len(posts))
	for i := range posts {
		postPtrs[i] = &posts[i]
	}
	c.signEventPostResponse(ctx.Request.Context(), postPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// DeletePost deletes a discussion post
func (c *EventController) DeletePost(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	postID, err := primitive.ObjectIDFromHex(ctx.Param("postId"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid post ID")
		return
	}

	if err := c.eventService.DeletePost(ctx, eventID, postID, userID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

// ReactToPost adds a reaction to a post
func (c *EventController) ReactToPost(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	postID, err := primitive.ObjectIDFromHex(ctx.Param("postId"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid post ID")
		return
	}

	var req models.ReactToPostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.eventService.ReactToPost(ctx, postID, userID, req.Emoji); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ================================
// Attendees Endpoints
// ================================

// GetAttendees returns attendees for an event
func (c *EventController) GetAttendees(ctx *gin.Context) {
	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	status := models.RSVPStatus(ctx.Query("status"))
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "20"), 10, 64)

	response, err := c.eventService.GetAttendees(ctx, eventID, status, limit, page)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	c.signAttendeeResponse(ctx.Request.Context(), response)

	ctx.JSON(http.StatusOK, response)
}

// ================================
// Co-Host Endpoints
// ================================

// AddCoHost adds a co-host to an event
func (c *EventController) AddCoHost(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.AddCoHostRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	coHostID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := c.eventService.AddCoHost(ctx, eventID, userID, coHostID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// RemoveCoHost removes a co-host from an event
func (c *EventController) RemoveCoHost(ctx *gin.Context) {
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	coHostID, err := primitive.ObjectIDFromHex(ctx.Param("userId"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := c.eventService.RemoveCoHost(ctx, eventID, userID, coHostID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.Status(http.StatusOK)
}

// ================================
// Categories Endpoint
// ================================

// GetCategories returns all event categories
func (c *EventController) GetCategories(ctx *gin.Context) {
	categories, err := c.eventService.GetCategories(ctx)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"categories": categories})
}

// ================================
// Search Endpoint
// ================================

// SearchEvents searches events with filters
func (c *EventController) SearchEvents(ctx *gin.Context) {
	userID, _ := utils.GetUserIDFromContext(ctx)

	var req models.SearchEventsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	events, total, err := c.eventService.SearchEvents(ctx, req, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	eventPtrs := make([]*models.EventResponse, len(events))
	for i := range events {
		eventPtrs[i] = &events[i]
	}
	c.signEventResponse(ctx.Request.Context(), eventPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   req.Page,
		"limit":  req.Limit,
	})
}

// ================================
// Share Endpoint
// ================================

// ShareEvent tracks an event share
func (c *EventController) ShareEvent(ctx *gin.Context) {
	eventID, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid event ID")
		return
	}

	if err := c.eventService.ShareEvent(ctx, eventID); err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

// ================================
// Nearby Events Endpoint
// ================================

// GetNearbyEvents returns events near a location
func (c *EventController) GetNearbyEvents(ctx *gin.Context) {
	userID, _ := utils.GetUserIDFromContext(ctx)

	lat, err := strconv.ParseFloat(ctx.Query("lat"), 64)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid latitude")
		return
	}

	lng, err := strconv.ParseFloat(ctx.Query("lng"), 64)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid longitude")
		return
	}

	radius, _ := strconv.ParseFloat(ctx.DefaultQuery("radius", "50"), 64)
	page, _ := strconv.ParseInt(ctx.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(ctx.DefaultQuery("limit", "20"), 10, 64)

	events, total, err := c.eventService.GetNearbyEvents(ctx, lat, lng, radius, limit, page, userID)
	if err != nil {
		utils.RespondWithError(ctx, utils.GetStatusCode(err), err.Error())
		return
	}

	// Sign URLs
	eventPtrs := make([]*models.EventResponse, len(events))
	for i := range events {
		eventPtrs[i] = &events[i]
	}
	c.signEventResponse(ctx, eventPtrs...)

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// ================================
// Signing Helpers
// ================================

func (c *EventController) signUserShort(ctx context.Context, u *models.UserShort) {
	if u.Avatar != "" {
		signed, err := c.storageClient.GetPresignedURL(ctx, u.Avatar, 15*time.Minute)
		if err == nil {
			u.Avatar = signed
		}
	}
}

func (c *EventController) signEvent(ctx context.Context, events ...*models.Event) {
	if len(events) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, e := range events {
		if e == nil {
			continue
		}
		wg.Add(1)
		go func(ev *models.Event) {
			defer wg.Done()
			if ev.CoverImage != "" {
				signed, err := c.storageClient.GetPresignedURL(ctx, ev.CoverImage, 15*time.Minute)
				if err == nil {
					ev.CoverImage = signed
				}
			}
		}(e)
	}
	wg.Wait()
}

func (c *EventController) signEventResponse(ctx context.Context, responses ...*models.EventResponse) {
	if len(responses) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, r := range responses {
		if r == nil {
			continue
		}
		wg.Add(1)
		go func(res *models.EventResponse) {
			defer wg.Done()
			var localWg sync.WaitGroup

			if res.CoverImage != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx, res.CoverImage, 15*time.Minute)
					if err == nil {
						res.CoverImage = signed
					}
				}()
			}

			// Creator Avatar
			localWg.Add(1)
			go func() {
				defer localWg.Done()
				c.signUserShort(ctx, &res.Creator)
			}()

			// Friends Going Avatars
			if len(res.FriendsGoing) > 0 {
				localWg.Add(len(res.FriendsGoing))
				for i := range res.FriendsGoing {
					go func(idx int) {
						defer localWg.Done()
						c.signUserShort(ctx, &res.FriendsGoing[idx])
					}(i)
				}
			}

			localWg.Wait()
		}(r)
	}
	wg.Wait()
}

func (c *EventController) signInvitationResponse(ctx context.Context, invitations ...*models.EventInvitationResponse) {
	if len(invitations) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, inv := range invitations {
		if inv == nil {
			continue
		}
		wg.Add(1)
		go func(i *models.EventInvitationResponse) {
			defer wg.Done()
			var localWg sync.WaitGroup

			// Event Cover Image
			if i.Event.CoverImage != "" {
				localWg.Add(1)
				go func() {
					defer localWg.Done()
					signed, err := c.storageClient.GetPresignedURL(ctx, i.Event.CoverImage, 15*time.Minute)
					if err == nil {
						i.Event.CoverImage = signed
					}
				}()
			}

			// Inviter Avatar
			localWg.Add(1)
			go func() {
				defer localWg.Done()
				c.signUserShort(ctx, &i.Inviter)
			}()

			localWg.Wait()
		}(inv)
	}
	wg.Wait()
}

func (c *EventController) signAttendeeResponse(ctx context.Context, response *models.AttendeesListResponse) {
	if response == nil || len(response.Attendees) == 0 {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(response.Attendees))
	for i := range response.Attendees {
		go func(idx int) {
			defer wg.Done()
			c.signUserShort(ctx, &response.Attendees[idx].User)
		}(i)
	}
	wg.Wait()
}

func (c *EventController) signEventPostResponse(ctx context.Context, posts ...*models.EventPostResponse) {
	if len(posts) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, p := range posts {
		if p == nil {
			continue
		}
		wg.Add(1)
		go func(post *models.EventPostResponse) {
			defer wg.Done()
			var localWg sync.WaitGroup

			// Media URLs
			if len(post.MediaURLs) > 0 {
				localWg.Add(len(post.MediaURLs))
				for i, url := range post.MediaURLs {
					go func(idx int, u string) {
						defer localWg.Done()
						if u == "" {
							return
						}
						signed, err := c.storageClient.GetPresignedURL(ctx, u, 15*time.Minute)
						if err == nil {
							post.MediaURLs[idx] = signed
						}
					}(i, url)
				}
			}

			// Author Avatar
			localWg.Add(1)
			go func() {
				defer localWg.Done()
				c.signUserShort(ctx, &post.Author)
			}()

			// Reactions Avatars
			if len(post.Reactions) > 0 {
				localWg.Add(len(post.Reactions))
				for i := range post.Reactions {
					go func(idx int) {
						defer localWg.Done()
						c.signUserShort(ctx, &post.Reactions[idx].User)
					}(i)
				}
			}

			localWg.Wait()
		}(p)
	}
	wg.Wait()
}

func (c *EventController) signBirthdayResponse(ctx context.Context, responses ...*models.BirthdayResponse) {
	if len(responses) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, r := range responses {
		if r == nil {
			continue
		}
		wg.Add(1)
		go func(res *models.BirthdayResponse) {
			defer wg.Done()
			var localWg sync.WaitGroup

			// Today Avatars
			if len(res.Today) > 0 {
				localWg.Add(len(res.Today))
				for i := range res.Today {
					go func(idx int) {
						defer localWg.Done()
						if res.Today[idx].Avatar != "" {
							signed, err := c.storageClient.GetPresignedURL(ctx, res.Today[idx].Avatar, 15*time.Minute)
							if err == nil {
								res.Today[idx].Avatar = signed
							}
						}
					}(i)
				}
			}

			// Upcoming Avatars
			if len(res.Upcoming) > 0 {
				localWg.Add(len(res.Upcoming))
				for i := range res.Upcoming {
					go func(idx int) {
						defer localWg.Done()
						if res.Upcoming[idx].Avatar != "" {
							signed, err := c.storageClient.GetPresignedURL(ctx, res.Upcoming[idx].Avatar, 15*time.Minute)
							if err == nil {
								res.Upcoming[idx].Avatar = signed
							}
						}
					}(i)
				}
			}

			localWg.Wait()
		}(r)
	}
	wg.Wait()
}
