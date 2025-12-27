package service

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/cache"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/pkg/async"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/producer"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/validation"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventCache abstracts cache operations for easier testing
type EventCache interface {
	SetEventStats(ctx context.Context, eventID string, stats *models.EventStats) error
	InvalidateUserRSVPStatus(ctx context.Context, userID, eventID string) error
	SetUserRSVPStatus(ctx context.Context, userID, eventID string, status models.RSVPStatus) error
	InvalidateFriendsGoing(ctx context.Context, userID, eventID string) error
	GetCategories(ctx context.Context) ([]models.EventCategory, error)
	SetCategories(ctx context.Context, categories []models.EventCategory) error
	GetTrendingEvents(ctx context.Context) ([]string, error)
	SetTrendingEvents(ctx context.Context, eventIDs []string) error
}

// cacheAdapter wraps the concrete cache implementation
type cacheAdapter struct {
	delegate *cache.EventCache
}

func NewEventCacheAdapter(c *cache.EventCache) EventCache {
	if c == nil {
		return nil
	}
	return &cacheAdapter{delegate: c}
}

func (a *cacheAdapter) SetEventStats(ctx context.Context, eventID string, stats *models.EventStats) error {
	return a.delegate.SetEventStats(ctx, eventID, stats)
}

func (a *cacheAdapter) InvalidateUserRSVPStatus(ctx context.Context, userID, eventID string) error {
	return a.delegate.InvalidateUserRSVPStatus(ctx, userID, eventID)
}

func (a *cacheAdapter) SetUserRSVPStatus(ctx context.Context, userID, eventID string, status models.RSVPStatus) error {
	return a.delegate.SetUserRSVPStatus(ctx, userID, eventID, status)
}

func (a *cacheAdapter) InvalidateFriendsGoing(ctx context.Context, userID, eventID string) error {
	return a.delegate.InvalidateFriendsGoing(ctx, userID, eventID)
}

func (a *cacheAdapter) GetCategories(ctx context.Context) ([]models.EventCategory, error) {
	return a.delegate.GetCategories(ctx)
}

func (a *cacheAdapter) SetCategories(ctx context.Context, categories []models.EventCategory) error {
	return a.delegate.SetCategories(ctx, categories)
}

func (a *cacheAdapter) GetTrendingEvents(ctx context.Context) ([]string, error) {
	return a.delegate.GetTrendingEvents(ctx)
}

func (a *cacheAdapter) SetTrendingEvents(ctx context.Context, eventIDs []string) error {
	return a.delegate.SetTrendingEvents(ctx, eventIDs)
}

const (
	asyncRetryAttempts = 5
	asyncRetryDelay    = time.Second
)

// EventBroadcaster defines interface for broadcasting event updates
type EventBroadcaster interface {
	BroadcastRSVP(event models.EventRSVPEvent)
	PublishEventUpdated(ctx context.Context, event models.EventUpdatedEvent)
	PublishEventDeleted(ctx context.Context, event models.EventDeletedEvent)
	PublishPostCreated(ctx context.Context, event models.EventPostCreatedEvent)
	PublishPostReaction(ctx context.Context, event models.EventPostReactionEvent)
	PublishInvitationUpdated(ctx context.Context, event models.EventInvitationUpdatedEvent)
	PublishCoHostAdded(ctx context.Context, event models.EventCoHostAddedEvent)
	PublishCoHostRemoved(ctx context.Context, event models.EventCoHostRemovedEvent)
}

type EventService struct {
	eventRepo            EventRepository
	userRepo             UserRepo
	eventGraphRepo       EventGraphRepo
	invitationRepo       InvitationRepo
	postRepo             PostRepo
	notificationProducer *producer.NotificationProducer
	eventCache           EventCache
	broadcaster          EventBroadcaster
	asyncRunner          *async.Runner // Added field
}

func NewEventService(
	eventRepo EventRepository,
	userRepo UserRepo,
	eventGraphRepo EventGraphRepo,
	invitationRepo InvitationRepo,
	postRepo PostRepo,
	notificationProducer *producer.NotificationProducer,
	eventCache EventCache,
	broadcaster EventBroadcaster,
	logger *slog.Logger, // Added parameter
) *EventService {
	return &EventService{
		eventRepo:            eventRepo,
		userRepo:             userRepo,
		eventGraphRepo:       eventGraphRepo,
		invitationRepo:       invitationRepo,
		postRepo:             postRepo,
		notificationProducer: notificationProducer,
		eventCache:           eventCache,
		broadcaster:          broadcaster,
		asyncRunner:          async.NewRunner(logger), // Initialized asyncRunner
	}
}

func (s *EventService) detachContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func (s *EventService) CreateEvent(ctx context.Context, userID primitive.ObjectID, req models.CreateEventRequest) (*models.Event, error) {
	// Validate request
	if err := validation.ValidateCreateEventRequest(&req); err != nil {
		return nil, err
	}

	event := &models.Event{
		Title:       req.Title,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Location:    req.Location,
		IsOnline:    req.IsOnline,
		Privacy:     req.Privacy,
		Category:    req.Category,
		CoverImage:  req.CoverImage,
		CreatorID:   userID,
	}

	// Creator is automatically going
	event.Attendees = []models.EventAttendee{
		{
			UserID:    userID,
			Status:    models.RSVPStatusGoing,
			Timestamp: time.Now(),
		},
	}
	event.Stats.GoingCount = 1

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	// Graph: Add Creator as Attendee
	if s.eventGraphRepo != nil {
		taskCtx := s.detachContext(ctx)
		s.asyncRunner.RunAsyncRetry(taskCtx, "add_creator_to_graph", func() error {
			graphCtx, cancel := context.WithTimeout(taskCtx, 5*time.Second)
			defer cancel()
			return s.eventGraphRepo.AddAttendee(graphCtx, userID, event.ID)
		}, asyncRetryAttempts, asyncRetryDelay)
	}

	return event, nil
}

func (s *EventService) GetEvent(ctx context.Context, id primitive.ObjectID, viewerID primitive.ObjectID) (*models.EventResponse, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Private event access control
	if event.Privacy == models.EventPrivacyPrivate {
		if !s.canAccessPrivateEvent(event, viewerID) {
			return nil, errors.New("unauthorized: you do not have access to this private event")
		}
	}

	return s.mapToResponse(ctx, event, viewerID)
}

// canAccessPrivateEvent checks if a viewer can access a private event
func (s *EventService) canAccessPrivateEvent(event *models.Event, viewerID primitive.ObjectID) bool {
	// Creator can always access
	if event.CreatorID == viewerID {
		return true
	}

	// Check if viewer is an attendee
	for _, attendee := range event.Attendees {
		if attendee.UserID == viewerID {
			return true
		}
	}

	// Check if viewer is a co-host
	for _, coHost := range event.CoHosts {
		if coHost.UserID == viewerID {
			return true
		}
	}

	// Check if viewer has an invitation
	if s.invitationRepo != nil {
		invitation, _ := s.invitationRepo.CheckExisting(context.Background(), event.ID, viewerID)
		if invitation != nil {
			return true
		}
	}

	return false
}

func (s *EventService) UpdateEvent(ctx context.Context, id, userID primitive.ObjectID, req models.UpdateEventRequest) (*models.EventResponse, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if event.CreatorID != userID {
		return nil, errors.New("unauthorized: only creator can update event")
	}

	// Validate request
	if err := validation.ValidateUpdateEventRequest(&req); err != nil {
		return nil, err
	}

	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.StartDate != nil {
		event.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		event.EndDate = *req.EndDate
	}
	if req.Location != "" {
		event.Location = req.Location
	}
	if req.IsOnline != nil {
		event.IsOnline = *req.IsOnline
	}
	if req.Privacy != "" {
		event.Privacy = req.Privacy
	}
	if req.Category != "" {
		event.Category = req.Category
	}
	if req.CoverImage != "" {
		event.CoverImage = req.CoverImage
	}

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	resp, err := s.mapToResponse(ctx, event, userID)
	if err == nil && s.broadcaster != nil {
		s.broadcaster.PublishEventUpdated(ctx, models.EventUpdatedEvent{
			ID:          event.ID.Hex(),
			Title:       event.Title,
			Description: event.Description,
			StartDate:   event.StartDate,
			EndDate:     event.EndDate,
			Location:    event.Location,
			IsOnline:    event.IsOnline,
			Privacy:     event.Privacy,
			Category:    event.Category,
			CoverImage:  event.CoverImage,
			UpdatedAt:   event.UpdatedAt,
		})
	}
	return resp, err
}

func (s *EventService) DeleteEvent(ctx context.Context, id, userID primitive.ObjectID) error {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if event.CreatorID != userID {
		return errors.New("unauthorized")
	}

	if err := s.eventRepo.Delete(ctx, id); err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishEventDeleted(ctx, models.EventDeletedEvent{
			ID:        id.Hex(),
			DeletedAt: time.Now(),
		})
	}

	return nil
}

func (s *EventService) ListEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64, query, category, period string) ([]models.EventResponse, int64, error) {
	filter := bson.M{}

	// Privacy and Visibility
	// Show Public events OR Friend events (if logic implemented) OR Events I created/attending
	// complex visibility logic. For now, let's just return Public events + My events.
	// Or simpler: Just return public events by default for Discover.
	filter["privacy"] = models.EventPrivacyPublic

	if query != "" {
		filter["$text"] = bson.M{"$search": query} // Assumes text index, fallback to regex if none
	}

	if category != "" {
		filter["category"] = category
	}

	// Period: today, week, weekend
	now := time.Now()
	if period == "today" {
		tomorrow := now.Add(24 * time.Hour)
		filter["start_date"] = bson.M{"$gte": now, "$lt": tomorrow}
	} else if period == "week" {
		nextWeek := now.Add(7 * 24 * time.Hour)
		filter["start_date"] = bson.M{"$gte": now, "$lt": nextWeek}
	} else if period == "past" {
		filter["start_date"] = bson.M{"$lt": now}
	} else {
		// Default upcoming
		filter["start_date"] = bson.M{"$gte": now}
	}

	events, total, err := s.eventRepo.List(ctx, limit, page, filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.EventResponse, len(events))
	for i, event := range events {
		resp, _ := s.mapToResponse(ctx, &event, userID)
		responses[i] = *resp
	}

	return responses, total, nil
}

func (s *EventService) GetUserEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.EventResponse, error) {
	events, err := s.eventRepo.GetUserEvents(ctx, userID, limit, page)
	if err != nil {
		return nil, err
	}

	responses := make([]models.EventResponse, len(events))
	for i, event := range events {
		resp, _ := s.mapToResponse(ctx, &event, userID)
		responses[i] = *resp
	}

	return responses, nil
}

func (s *EventService) GetFriendBirthdays(ctx context.Context, userID primitive.ObjectID) (*models.BirthdayResponse, error) {
	// 1. Get current user to find friends
	currentUser, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(currentUser.Friends) == 0 {
		return &models.BirthdayResponse{
			Today:    []models.BirthdayUser{},
			Upcoming: []models.BirthdayUser{},
		}, nil
	}

	// 2. Fetch friend birthdays efficiently
	todayUsers, upcomingUsers, err := s.userRepo.FindFriendBirthdays(ctx, currentUser.Friends)
	if err != nil {
		return nil, err
	}

	response := &models.BirthdayResponse{
		Today:    []models.BirthdayUser{},
		Upcoming: []models.BirthdayUser{},
	}

	now := time.Now()

	// process Today
	for _, f := range todayUsers {
		dob := *f.DateOfBirth
		age := now.Year() - dob.Year()
		// If today is birthday, age is exactly Year - Year

		response.Today = append(response.Today, models.BirthdayUser{
			ID:       f.ID.Hex(),
			Username: f.Username,
			FullName: f.FullName,
			Avatar:   f.Avatar,
			Age:      age,
			Date:     "Today",
		})
	}

	// process Upcoming
	for _, f := range upcomingUsers {
		dob := *f.DateOfBirth
		age := now.Year() - dob.Year()
		if now.YearDay() < dob.YearDay() {
			age--
		}
		// Age will be turning age? Usually users want "Turning X"
		// If birthday hasn't happened yet this year, they are age. On birthday they will be age+1.
		// Let's display the age they WILL be.
		age++

		// Format date
		thisYearBday := time.Date(now.Year(), dob.Month(), dob.Day(), 0, 0, 0, 0, now.Location())
		if thisYearBday.Before(now) {
			thisYearBday = thisYearBday.AddDate(1, 0, 0)
		}

		response.Upcoming = append(response.Upcoming, models.BirthdayUser{
			ID:       f.ID.Hex(),
			Username: f.Username,
			FullName: f.FullName,
			Avatar:   f.Avatar,
			Age:      age,
			Date:     thisYearBday.Format("January 02"),
		})
	}

	return response, nil
}

func (s *EventService) RSVP(ctx context.Context, eventID primitive.ObjectID, userID primitive.ObjectID, status models.RSVPStatus) error {
	_, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	// Update RSVP
	attendee := models.EventAttendee{
		UserID:    userID,
		Status:    status,
		Timestamp: time.Now(),
	}

	if err := s.eventRepo.AddOrUpdateAttendee(ctx, eventID, attendee); err != nil {
		return err
	}

	// Recalculate stats
	// This is heavy but mostly accurate.
	// Optimally we'd do this incrementally or async.
	updatedEvent, _ := s.eventRepo.GetByID(ctx, eventID)
	if updatedEvent != nil {
		var going, interested, invited int64
		for _, a := range updatedEvent.Attendees {
			switch a.Status {
			case models.RSVPStatusGoing:
				going++
			case models.RSVPStatusInterested:
				interested++
			case models.RSVPStatusInvited:
				invited++
			}
		}
		stats := models.EventStats{
			GoingCount:      going,
			InterestedCount: interested,
			InvitedCount:    invited,
			ShareCount:      updatedEvent.Stats.ShareCount,
		}
		s.eventRepo.UpdateStats(ctx, eventID, stats)

		// Cache the updated stats
		if s.eventCache != nil {
			s.eventCache.SetEventStats(ctx, eventID.Hex(), &stats)
			// Invalidate user's RSVP status cache
			s.eventCache.InvalidateUserRSVPStatus(ctx, userID.Hex(), eventID.Hex())
			// Cache the new RSVP status
			s.eventCache.SetUserRSVPStatus(ctx, userID.Hex(), eventID.Hex(), status)
			s.invalidateFriendsGoing(ctx, eventID, userID)
		}

		// Broadcast RSVP update
		if s.broadcaster != nil {
			s.broadcaster.BroadcastRSVP(models.EventRSVPEvent{
				EventID:   eventID.Hex(),
				UserID:    userID.Hex(),
				Status:    status,
				Timestamp: time.Now(),
				Stats:     stats,
			})
		}
	}

	// Update Graph (Async)
	if s.eventGraphRepo != nil {
		taskCtx := s.detachContext(ctx)
		s.asyncRunner.RunAsyncRetry(taskCtx, "update_rsvp_graph", func() error {
			graphCtx, cancel := context.WithTimeout(taskCtx, 5*time.Second)
			defer cancel()
			if status == models.RSVPStatusGoing {
				return s.eventGraphRepo.AddAttendee(graphCtx, userID, eventID)
			}
			return s.eventGraphRepo.RemoveAttendee(graphCtx, userID, eventID)
		}, asyncRetryAttempts, asyncRetryDelay)
	}

	return nil
}

func (s *EventService) mapToResponse(ctx context.Context, event *models.Event, viewerID primitive.ObjectID) (*models.EventResponse, error) {
	// Fetch Creator info
	creator, _ := s.userRepo.FindByID(ctx, event.CreatorID)
	creatorShort := models.UserShort{
		ID:       event.CreatorID.Hex(),
		Username: "Unknown",
	}
	if creator != nil {
		creatorShort.Username = creator.Username
		creatorShort.FullName = creator.FullName
		creatorShort.Avatar = creator.Avatar
	}

	// Determine MyStatus and IsHost
	var myStatus models.RSVPStatus
	for _, attendee := range event.Attendees {
		if attendee.UserID == viewerID {
			myStatus = attendee.Status
			break
		}
	}

	// Fetch friends going (from Neo4j)
	var friendsGoing []models.UserShort
	if s.eventGraphRepo != nil && !viewerID.IsZero() {
		friendIDs, err := s.eventGraphRepo.GetFriendsGoing(ctx, viewerID, event.ID)
		if err == nil && len(friendIDs) > 0 {
			// Limit to first 5 friends for display
			limit := 5
			if len(friendIDs) < limit {
				limit = len(friendIDs)
			}

			// Batch fetch friend details
			var friendOIDs []primitive.ObjectID
			for i := 0; i < limit; i++ {
				if oid, err := primitive.ObjectIDFromHex(friendIDs[i]); err == nil {
					friendOIDs = append(friendOIDs, oid)
				}
			}

			if len(friendOIDs) > 0 {
				friends, err := s.userRepo.FindByIDs(ctx, friendOIDs)
				if err == nil {
					for _, friend := range friends {
						friendsGoing = append(friendsGoing, models.UserShort{
							ID:       friend.ID.Hex(),
							Username: friend.Username,
							FullName: friend.FullName,
							Avatar:   friend.Avatar,
						})
					}
				}
			}
		}
	}

	return &models.EventResponse{
		ID:           event.ID.Hex(),
		Title:        event.Title,
		Description:  event.Description,
		StartDate:    event.StartDate,
		EndDate:      event.EndDate,
		Location:     event.Location,
		IsOnline:     event.IsOnline,
		Privacy:      event.Privacy,
		Category:     event.Category,
		CoverImage:   event.CoverImage,
		Creator:      creatorShort,
		Stats:        event.Stats,
		MyStatus:     myStatus,
		IsHost:       event.CreatorID == viewerID,
		FriendsGoing: friendsGoing,
		CreatedAt:    event.CreatedAt,
	}, nil
}

func (s *EventService) invalidateFriendsGoing(ctx context.Context, eventID, userID primitive.ObjectID) {
	if s.eventCache == nil {
		return
	}

	eventHex := eventID.Hex()
	seen := make(map[string]struct{})
	add := func(id primitive.ObjectID) {
		if id.IsZero() {
			return
		}
		hex := id.Hex()
		if _, exists := seen[hex]; exists {
			return
		}
		seen[hex] = struct{}{}
		_ = s.eventCache.InvalidateFriendsGoing(ctx, hex, eventHex)
	}

	add(userID)

	if event, err := s.eventRepo.GetByID(ctx, eventID); err == nil && event != nil {
		for _, attendee := range event.Attendees {
			add(attendee.UserID)
		}
	}

	if user, err := s.userRepo.FindByID(ctx, userID); err == nil && user != nil {
		for _, fid := range user.Friends {
			add(fid)
		}
	}
}

// ===============================
// Invitation Methods
// ===============================

// InviteFriends sends invitations to multiple friends
func (s *EventService) InviteFriends(ctx context.Context, eventID, inviterID primitive.ObjectID, friendIDs []string, message string) error {
	// Verify event exists and inviter has permission
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	// Only creator, co-hosts, or going attendees can invite
	canInvite := event.CreatorID == inviterID
	if !canInvite {
		for _, coHost := range event.CoHosts {
			if coHost.UserID == inviterID {
				canInvite = true
				break
			}
		}
	}
	if !canInvite {
		for _, attendee := range event.Attendees {
			if attendee.UserID == inviterID && attendee.Status == models.RSVPStatusGoing {
				canInvite = true
				break
			}
		}
	}
	if !canInvite {
		return errors.New("unauthorized: you cannot invite to this event")
	}

	// Create invitations
	var invitations []models.EventInvitation
	for _, friendIDStr := range friendIDs {
		friendID, err := primitive.ObjectIDFromHex(friendIDStr)
		if err != nil {
			continue
		}

		// Check if already invited
		existing, _ := s.invitationRepo.CheckExisting(ctx, eventID, friendID)
		if existing != nil {
			continue
		}

		// Check if already attending
		isAttendee := false
		for _, a := range event.Attendees {
			if a.UserID == friendID {
				isAttendee = true
				break
			}
		}
		if isAttendee {
			continue
		}

		invitations = append(invitations, models.EventInvitation{
			EventID:   eventID,
			InviterID: inviterID,
			InviteeID: friendID,
			Message:   message,
		})
	}

	if len(invitations) > 0 {
		err := s.invitationRepo.CreateMany(ctx, invitations)
		if err != nil {
			return err
		}

		// Create notifications for each invitee
		if s.notificationProducer != nil {
			inviter, _ := s.userRepo.FindByID(ctx, inviterID)
			inviterUsername := "Someone"
			inviterAvatar := ""
			if inviter != nil {
				inviterUsername = inviter.Username
				inviterAvatar = inviter.Avatar
			}

			taskCtx := s.detachContext(ctx)
			for _, inv := range invitations {
				notification := &models.Notification{
					ID:          primitive.NewObjectID(),
					RecipientID: inv.InviteeID,
					SenderID:    inviterID,
					Type:        models.NotificationTypeEventInvite,
					TargetID:    eventID,
					TargetType:  "event",
					Content:     inviterUsername + " invited you to " + event.Title,
					Data: map[string]interface{}{
						"event_id":        eventID.Hex(),
						"event_title":     event.Title,
						"sender_id":       inviterID.Hex(),
						"sender_username": inviterUsername,
						"sender_avatar":   inviterAvatar,
					},
					Read:      false,
					CreatedAt: time.Now(),
				}

				s.asyncRunner.RunAsyncRetry(taskCtx, "publish_invite_notification", func() error {
					notifyCtx, cancel := context.WithTimeout(taskCtx, 5*time.Second)
					defer cancel()
					return s.notificationProducer.PublishNotification(notifyCtx, notification)
				}, asyncRetryAttempts, asyncRetryDelay)
			}
		}
	}

	return nil
}

// GetUserInvitations returns pending invitations for a user
func (s *EventService) GetUserInvitations(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.EventInvitationResponse, int64, error) {
	invitations, total, err := s.invitationRepo.GetUserInvitations(ctx, userID, models.InvitationStatusPending, limit, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.EventInvitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		// Get event info
		event, err := s.eventRepo.GetByID(ctx, inv.EventID)
		if err != nil {
			continue
		}

		// Get inviter info
		inviter, _ := s.userRepo.FindByID(ctx, inv.InviterID)
		inviterShort := models.UserShort{ID: inv.InviterID.Hex(), Username: "Unknown"}
		if inviter != nil {
			inviterShort.Username = inviter.Username
			inviterShort.FullName = inviter.FullName
			inviterShort.Avatar = inviter.Avatar
		}

		responses = append(responses, models.EventInvitationResponse{
			ID: inv.ID.Hex(),
			Event: models.EventShort{
				ID:         event.ID.Hex(),
				Title:      event.Title,
				CoverImage: event.CoverImage,
				StartDate:  event.StartDate,
				Location:   event.Location,
			},
			Inviter:   inviterShort,
			Status:    inv.Status,
			Message:   inv.Message,
			CreatedAt: inv.CreatedAt,
		})
	}

	return responses, total, nil
}

// RespondToInvitation accepts or declines an invitation
func (s *EventService) RespondToInvitation(ctx context.Context, invitationID, userID primitive.ObjectID, accept bool) error {
	invitation, err := s.invitationRepo.GetByID(ctx, invitationID)
	if err != nil {
		return err
	}

	if invitation.InviteeID != userID {
		return errors.New("unauthorized")
	}

	if invitation.Status != models.InvitationStatusPending {
		return errors.New("invitation already responded")
	}

	// Get event info for notification
	event, err := s.eventRepo.GetByID(ctx, invitation.EventID)
	if err != nil {
		return err
	}

	var newStatus models.EventInvitationStatus
	if accept {
		newStatus = models.InvitationStatusAccepted
		// Add user as going
		if err := s.RSVP(ctx, invitation.EventID, userID, models.RSVPStatusGoing); err != nil {
			return err
		}
	} else {
		newStatus = models.InvitationStatusDeclined
	}

	if err := s.invitationRepo.UpdateStatus(ctx, invitationID, newStatus); err != nil {
		return err
	}

	// Broadcast Invitation Update
	if s.broadcaster != nil {
		s.broadcaster.PublishInvitationUpdated(ctx, models.EventInvitationUpdatedEvent{
			InvitationID: invitationID.Hex(),
			EventID:      invitation.EventID.Hex(),
			InviteeID:    userID.Hex(),
			Status:       newStatus,
			Timestamp:    time.Now(),
		})
	}

	// Create notification for the inviter
	if s.notificationProducer != nil {
		invitee, _ := s.userRepo.FindByID(ctx, userID)
		inviteeUsername := "Someone"
		inviteeAvatar := ""
		if invitee != nil {
			inviteeUsername = invitee.Username
			inviteeAvatar = invitee.Avatar
		}

		var notificationType models.NotificationType
		var content string
		if accept {
			notificationType = models.NotificationTypeEventInviteAccepted
			content = inviteeUsername + " accepted your invitation to " + event.Title
		} else {
			notificationType = models.NotificationTypeEventInviteDeclined
			content = inviteeUsername + " declined your invitation to " + event.Title
		}

		notification := &models.Notification{
			ID:          primitive.NewObjectID(),
			RecipientID: invitation.InviterID,
			SenderID:    userID,
			Type:        notificationType,
			TargetID:    invitation.EventID,
			TargetType:  "event",
			Content:     content,
			Data: map[string]interface{}{
				"event_id":        invitation.EventID.Hex(),
				"event_title":     event.Title,
				"sender_id":       userID.Hex(),
				"sender_username": inviteeUsername,
				"sender_avatar":   inviteeAvatar,
				"accepted":        accept,
			},
			Read:      false,
			CreatedAt: time.Now(),
		}
		taskCtx := s.detachContext(ctx)
		s.asyncRunner.RunAsyncRetry(taskCtx, "publish_response_notification", func() error {
			notifyCtx, cancel := context.WithTimeout(taskCtx, 5*time.Second)
			defer cancel()
			return s.notificationProducer.PublishNotification(notifyCtx, notification)
		}, asyncRetryAttempts, asyncRetryDelay)
	}

	return nil
}

// ===============================
// Discussion/Post Methods
// ===============================

// CreatePost creates a discussion post on an event
func (s *EventService) CreatePost(ctx context.Context, eventID, authorID primitive.ObjectID, req models.CreateEventPostRequest) (*models.EventPostResponse, error) {
	// Verify event exists
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Check if user can post (attendee or host)
	canPost := event.CreatorID == authorID
	if !canPost {
		for _, a := range event.Attendees {
			if a.UserID == authorID && a.Status != models.RSVPStatusNotGoing {
				canPost = true
				break
			}
		}
	}
	if !canPost {
		return nil, errors.New("only attendees can post in event discussions")
	}

	post := &models.EventPost{
		EventID:   eventID,
		AuthorID:  authorID,
		Content:   req.Content,
		MediaURLs: req.MediaURLs,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	// Get author info for response
	author, _ := s.userRepo.FindByID(ctx, authorID)
	authorShort := models.UserShort{ID: authorID.Hex(), Username: "Unknown"}
	if author != nil {
		authorShort.Username = author.Username
		authorShort.FullName = author.FullName
		authorShort.Avatar = author.Avatar
	}

	resp := &models.EventPostResponse{
		ID:        post.ID.Hex(),
		Author:    authorShort,
		Content:   post.Content,
		MediaURLs: post.MediaURLs,
		Reactions: []models.EventPostReactionResponse{},
		CreatedAt: post.CreatedAt,
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishPostCreated(ctx, models.EventPostCreatedEvent{
			Post:    *resp,
			EventID: eventID.Hex(),
		})
	}

	return resp, nil
}

// GetPosts returns discussion posts for an event
func (s *EventService) GetPosts(ctx context.Context, eventID primitive.ObjectID, limit, page int64) ([]models.EventPostResponse, int64, error) {
	posts, total, err := s.postRepo.GetByEventID(ctx, eventID, limit, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.EventPostResponse, 0, len(posts))
	for _, post := range posts {
		// Get author info
		author, _ := s.userRepo.FindByID(ctx, post.AuthorID)
		authorShort := models.UserShort{ID: post.AuthorID.Hex(), Username: "Unknown"}
		if author != nil {
			authorShort.Username = author.Username
			authorShort.FullName = author.FullName
			authorShort.Avatar = author.Avatar
		}

		// Map reactions
		reactions := make([]models.EventPostReactionResponse, 0, len(post.Reactions))
		for _, r := range post.Reactions {
			user, _ := s.userRepo.FindByID(ctx, r.UserID)
			userShort := models.UserShort{ID: r.UserID.Hex()}
			if user != nil {
				userShort.Username = user.Username
				userShort.Avatar = user.Avatar
			}
			reactions = append(reactions, models.EventPostReactionResponse{
				User:      userShort,
				Emoji:     r.Emoji,
				Timestamp: r.Timestamp,
			})
		}

		responses = append(responses, models.EventPostResponse{
			ID:        post.ID.Hex(),
			Author:    authorShort,
			Content:   post.Content,
			MediaURLs: post.MediaURLs,
			Reactions: reactions,
			CreatedAt: post.CreatedAt,
		})
	}

	return responses, total, nil
}

// DeletePost deletes a discussion post
func (s *EventService) DeletePost(ctx context.Context, eventID, postID, userID primitive.ObjectID) error {
	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}

	if post.EventID != eventID {
		return errors.New("post does not belong to this event")
	}

	// Only author or event host can delete
	event, _ := s.eventRepo.GetByID(ctx, eventID)
	if post.AuthorID != userID && event.CreatorID != userID {
		return errors.New("unauthorized")
	}

	return s.postRepo.Delete(ctx, postID)
}

// ReactToPost adds or updates a reaction on a post
func (s *EventService) ReactToPost(ctx context.Context, postID, userID primitive.ObjectID, emoji string) error {
	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}

	reaction := models.EventPostReaction{
		UserID:    userID,
		Emoji:     emoji,
		Timestamp: time.Now(),
	}

	if err := s.postRepo.AddReaction(ctx, postID, reaction); err != nil {
		return err
	}

	// Fetch user for response
	user, _ := s.userRepo.FindByID(ctx, userID)
	userShort := models.UserShort{ID: userID.Hex(), Username: "Unknown"}
	if user != nil {
		userShort.Username = user.Username
		userShort.Avatar = user.Avatar
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishPostReaction(ctx, models.EventPostReactionEvent{
			PostID:    postID.Hex(),
			EventID:   post.EventID.Hex(),
			User:      userShort,
			Emoji:     emoji,
			Timestamp: reaction.Timestamp,
		})
	}

	return nil
}

// ===============================
// Attendees Methods
// ===============================

// GetAttendees returns attendees for an event with pagination
func (s *EventService) GetAttendees(ctx context.Context, eventID primitive.ObjectID, status models.RSVPStatus, limit, page int64) (*models.AttendeesListResponse, error) {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	attendees, total, err := s.eventRepo.GetAttendeesByStatus(ctx, eventID, status, limit, page)
	if err != nil {
		return nil, err
	}

	responses := make([]models.EventAttendeeResponse, 0, len(attendees))
	for _, a := range attendees {
		user, _ := s.userRepo.FindByID(ctx, a.UserID)
		userShort := models.UserShort{ID: a.UserID.Hex(), Username: "Unknown"}
		if user != nil {
			userShort.Username = user.Username
			userShort.FullName = user.FullName
			userShort.Avatar = user.Avatar
		}

		isCoHost := false
		for _, ch := range event.CoHosts {
			if ch.UserID == a.UserID {
				isCoHost = true
				break
			}
		}

		responses = append(responses, models.EventAttendeeResponse{
			User:      userShort,
			Status:    a.Status,
			Timestamp: a.Timestamp,
			IsHost:    event.CreatorID == a.UserID,
			IsCoHost:  isCoHost,
		})
	}

	return &models.AttendeesListResponse{
		Attendees: responses,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// ===============================
// Co-Host Methods
// ===============================

// AddCoHost adds a co-host to an event
func (s *EventService) AddCoHost(ctx context.Context, eventID, userID, coHostID primitive.ObjectID) error {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.CreatorID != userID {
		return errors.New("only the event creator can add co-hosts")
	}

	// Check if user is already a co-host
	for _, ch := range event.CoHosts {
		if ch.UserID == coHostID {
			return errors.New("user is already a co-host")
		}
	}

	coHost := models.EventCoHost{
		UserID:    coHostID,
		AddedAt:   time.Now(),
		AddedByID: userID,
	}

	if err := s.eventRepo.AddCoHost(ctx, eventID, coHost); err != nil {
		return err
	}

	if s.broadcaster != nil {
		coHostUser, _ := s.userRepo.FindByID(ctx, coHostID)
		coHostShort := models.UserShort{ID: coHostID.Hex()}
		if coHostUser != nil {
			coHostShort.Username = coHostUser.Username
			coHostShort.Avatar = coHostUser.Avatar
		}

		s.broadcaster.PublishCoHostAdded(ctx, models.EventCoHostAddedEvent{
			EventID:   eventID.Hex(),
			CoHost:    coHostShort,
			AddedBy:   userID.Hex(),
			Timestamp: coHost.AddedAt,
		})
	}

	return nil
}

// RemoveCoHost removes a co-host from an event
func (s *EventService) RemoveCoHost(ctx context.Context, eventID, userID, coHostID primitive.ObjectID) error {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.CreatorID != userID {
		return errors.New("only the event creator can remove co-hosts")
	}

	if err := s.eventRepo.RemoveCoHost(ctx, eventID, coHostID); err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.PublishCoHostRemoved(ctx, models.EventCoHostRemovedEvent{
			EventID:   eventID.Hex(),
			CoHostID:  coHostID.Hex(),
			RemovedBy: userID.Hex(),
			Timestamp: time.Now(),
		})
	}

	return nil
}

// ===============================
// Categories Methods
// ===============================

// GetCategories returns all event categories with counts
func (s *EventService) GetCategories(ctx context.Context) ([]models.EventCategory, error) {
	if s.eventCache != nil {
		if categories, err := s.eventCache.GetCategories(ctx); err == nil && len(categories) > 0 {
			return categories, nil
		}
	}

	categories, err := s.eventRepo.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	if s.eventCache != nil {
		if err := s.eventCache.SetCategories(ctx, categories); err != nil {
			log.Printf("failed to cache event categories: %v", err)
		}
	}
	return categories, nil
}

// ===============================
// Share Methods
// ===============================

// ShareEvent increments share count
func (s *EventService) ShareEvent(ctx context.Context, eventID primitive.ObjectID) error {
	return s.eventRepo.IncrementShareCount(ctx, eventID)
}

// ===============================
// Search Methods
// ===============================

// SearchEvents searches events with filters
func (s *EventService) SearchEvents(ctx context.Context, req models.SearchEventsRequest, userID primitive.ObjectID) ([]models.EventResponse, int64, error) {
	filter := bson.M{}

	// Privacy filter - show public events
	filter["privacy"] = models.EventPrivacyPublic

	// Category filter
	if req.Category != "" {
		filter["category"] = req.Category
	}

	// Online filter
	if req.Online != nil {
		filter["is_online"] = *req.Online
	}

	// Period filter
	now := time.Now()
	switch req.Period {
	case "today":
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		filter["start_date"] = bson.M{"$gte": now, "$lt": tomorrow}
	case "tomorrow":
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		dayAfter := tomorrow.AddDate(0, 0, 1)
		filter["start_date"] = bson.M{"$gte": tomorrow, "$lt": dayAfter}
	case "this_week":
		nextWeek := now.AddDate(0, 0, 7)
		filter["start_date"] = bson.M{"$gte": now, "$lt": nextWeek}
	case "this_weekend":
		// Find next Saturday
		daysUntilSat := (6 - int(now.Weekday()) + 7) % 7
		if daysUntilSat == 0 && now.Hour() >= 12 {
			daysUntilSat = 7
		}
		saturday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSat, 0, 0, 0, 0, now.Location())
		monday := saturday.AddDate(0, 0, 2)
		filter["start_date"] = bson.M{"$gte": saturday, "$lt": monday}
	default:
		// Default to upcoming
		filter["start_date"] = bson.M{"$gte": now}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	events, total, err := s.eventRepo.Search(ctx, req.Query, filter, limit, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.EventResponse, 0, len(events))
	for _, event := range events {
		resp, _ := s.mapToResponse(ctx, &event, userID)
		if resp != nil {
			responses = append(responses, *resp)
		}
	}

	return responses, total, nil
}

// GetNearbyEvents returns events near a location
func (s *EventService) GetNearbyEvents(ctx context.Context, lat, lng, radiusKm float64, limit, page int64, userID primitive.ObjectID) ([]models.EventResponse, int64, error) {
	if radiusKm <= 0 {
		radiusKm = 50 // Default 50km radius
	}

	events, total, err := s.eventRepo.GetNearbyEvents(ctx, lat, lng, radiusKm, limit, page)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]models.EventResponse, 0, len(events))
	for _, event := range events {
		resp, _ := s.mapToResponse(ctx, &event, userID)
		if resp != nil {
			responses = append(responses, *resp)
		}
	}

	return responses, total, nil
}
