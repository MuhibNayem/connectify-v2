package service

import (
	"context"
	"sort"
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventRecommendationService provides event recommendations based on social graph
type EventRecommendationService struct {
	eventRepo      EventRepository
	eventGraphRepo EventGraphRepo
	userRepo       UserRepo
	friendshipRepo FriendshipRepo
	eventCache     EventCache
}

// NewEventRecommendationService creates a new recommendation service
func NewEventRecommendationService(
	eventRepo EventRepository,
	eventGraphRepo EventGraphRepo,
	userRepo UserRepo,
	friendshipRepo FriendshipRepo,
	eventCache EventCache,
) *EventRecommendationService {
	return &EventRecommendationService{
		eventRepo:      eventRepo,
		eventGraphRepo: eventGraphRepo,
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
		eventCache:     eventCache,
	}
}

// EventRecommendation represents a recommended event with score
type EventRecommendation struct {
	EventID      string             `json:"event_id"`
	Score        float64            `json:"score"`
	FriendsGoing []models.UserShort `json:"friends_going"`
	FriendCount  int                `json:"friend_count"`
	Reason       string             `json:"reason"`
	Event        *models.Event      `json:"event,omitempty"`
}

// GetRecommendations returns personalized event recommendations for a user
func (s *EventRecommendationService) GetRecommendations(ctx context.Context, userID primitive.ObjectID, limit int) ([]EventRecommendation, error) {
	// 1. Get user's friends
	// 1. Get user's friend IDs
	friendIDs, err := s.friendshipRepo.GetFriends(ctx, userID)
	if err != nil {
		return s.getPopularEvents(ctx, limit)
	}

	if len(friendIDs) == 0 {
		return s.getPopularEvents(ctx, limit)
	}

	// Fetch friend details from User Local Repo
	friends, err := s.userRepo.FindByIDs(ctx, friendIDs)
	if err != nil {
		// If fails, we can just proceed with empty friends map or return popular
		return s.getPopularEvents(ctx, limit)
	}

	friendMap := make(map[string]models.UserShort)
	for _, f := range friends {
		friendMap[f.ID.Hex()] = models.UserShort{
			ID:       f.ID.Hex(),
			Username: f.Username,
			FullName: f.FullName,
			Avatar:   f.Avatar,
		}
	}

	if len(friendIDs) == 0 {
		return s.getPopularEvents(ctx, limit)
	}

	// 2. Get upcoming public events
	filter := bson.M{
		"privacy":    models.EventPrivacyPublic,
		"start_date": bson.M{"$gt": time.Now()},
	}
	events, _, err := s.eventRepo.List(ctx, 100, 1, filter)
	if err != nil {
		return nil, err
	}

	// 3. Score events based on how many friends are attending
	recommendations := []EventRecommendation{}

	for _, event := range events {
		// Skip events user is already attending
		isAttending := false
		for _, a := range event.Attendees {
			if a.UserID == userID {
				isAttending = true
				break
			}
		}
		if isAttending {
			continue
		}

		// Count friends attending this event
		friendsGoing := []models.UserShort{}
		for _, attendee := range event.Attendees {
			if attendee.Status == models.RSVPStatusGoing || attendee.Status == models.RSVPStatusInterested {
				if friend, exists := friendMap[attendee.UserID.Hex()]; exists {
					friendsGoing = append(friendsGoing, friend)
				}
			}
		}

		if len(friendsGoing) == 0 {
			continue // Only recommend events with friends going
		}

		// Calculate score
		score := float64(len(friendsGoing)) * 2.0
		score += float64(event.Stats.GoingCount) * 0.1

		e := event
		recommendations = append(recommendations, EventRecommendation{
			EventID:      event.ID.Hex(),
			Score:        score,
			FriendsGoing: friendsGoing,
			FriendCount:  len(friendsGoing),
			Reason:       s.buildReason(len(friendsGoing)),
			Event:        &e,
		})
	}

	// 4. Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// 5. Limit results
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

// GetGraphBasedRecommendations uses Neo4j graph for FB-scale recommendations
// with automatic fallback to MongoDB if graph is unavailable
func (s *EventRecommendationService) GetGraphBasedRecommendations(ctx context.Context, userID primitive.ObjectID, limit int) ([]EventRecommendation, error) {
	// Try graph-based recommendations first
	graphRecs, err := s.eventGraphRepo.GetRecommendedEventsFromGraph(ctx, userID.Hex(), limit*2)
	if err != nil {
		// Fallback to MongoDB-based recommendations
		return s.GetRecommendations(ctx, userID, limit)
	}

	if len(graphRecs) == 0 {
		return s.getPopularEvents(ctx, limit)
	}

	// Fetch event details for graph recommendations
	recommendations := make([]EventRecommendation, 0, len(graphRecs))
	for _, gr := range graphRecs {
		eventID, err := primitive.ObjectIDFromHex(gr.EventID)
		if err != nil {
			continue
		}

		event, err := s.eventRepo.GetByID(ctx, eventID)
		if err != nil {
			continue
		}

		// Skip if user is already attending
		isAttending := false
		for _, a := range event.Attendees {
			if a.UserID == userID {
				isAttending = true
				break
			}
		}
		if isAttending {
			continue
		}

		// Build friend info for display
		friendsGoing := make([]models.UserShort, 0, len(gr.FriendsGoing))
		for _, fid := range gr.FriendsGoing {
			objID, err := primitive.ObjectIDFromHex(fid)
			if err != nil {
				continue
			}
			u, err := s.userRepo.FindByID(ctx, objID)
			if err == nil && u != nil {
				friendsGoing = append(friendsGoing, models.UserShort{
					ID:       u.ID.Hex(),
					Username: u.Username,
					FullName: u.FullName,
					Avatar:   u.Avatar,
				})
			}
		}

		reason := s.buildGraphReason(len(gr.FriendsGoing), len(gr.FoFGoing), gr.CategoryMatch)

		recommendations = append(recommendations, EventRecommendation{
			EventID:      gr.EventID,
			Score:        gr.Score,
			FriendsGoing: friendsGoing,
			FriendCount:  len(gr.FriendsGoing),
			Reason:       reason,
			Event:        event,
		})
	}

	// Limit final results
	if len(recommendations) > limit {
		recommendations = recommendations[:limit]
	}

	return recommendations, nil
}

func (s *EventRecommendationService) buildGraphReason(friendCount, fofCount int, categoryMatch bool) string {
	parts := []string{}
	if friendCount == 1 {
		parts = append(parts, "1 friend is going")
	} else if friendCount > 1 {
		parts = append(parts, "friends are going")
	}
	if fofCount > 0 {
		parts = append(parts, "friends of friends attending")
	}
	if categoryMatch {
		parts = append(parts, "matches your interests")
	}
	if len(parts) == 0 {
		return "Recommended for you"
	}
	return parts[0] // Return primary reason
}

func (s *EventRecommendationService) buildReason(friendCount int) string {
	if friendCount == 1 {
		return "1 friend is going"
	}
	return "friends are going"
}

func (s *EventRecommendationService) getPopularEvents(ctx context.Context, limit int) ([]EventRecommendation, error) {
	// Get upcoming public events sorted by popularity
	filter := bson.M{
		"privacy":    models.EventPrivacyPublic,
		"start_date": bson.M{"$gt": time.Now()},
	}
	events, _, err := s.eventRepo.List(ctx, int64(limit), 1, filter)
	if err != nil {
		return nil, err
	}

	recommendations := []EventRecommendation{}
	for _, event := range events {
		score := float64(event.Stats.GoingCount) + float64(event.Stats.InterestedCount)*0.5
		e := event
		recommendations = append(recommendations, EventRecommendation{
			EventID: event.ID.Hex(),
			Score:   score,
			Reason:  "Popular event",
			Event:   &e,
		})
	}

	return recommendations, nil
}

// TrendingScore represents an event's trending score
type TrendingScore struct {
	EventID string        `json:"event_id"`
	Score   float64       `json:"score"`
	Event   *models.Event `json:"event,omitempty"`
}

// GetTrendingEvents returns trending events based on recent activity
func (s *EventRecommendationService) GetTrendingEvents(ctx context.Context, limit int) ([]TrendingScore, error) {
	// Try cache first
	if s.eventCache != nil {
		cached, err := s.eventCache.GetTrendingEvents(ctx)
		if err == nil && len(cached) > 0 {
			scores := []TrendingScore{}
			for _, id := range cached {
				scores = append(scores, TrendingScore{EventID: id})
			}
			if len(scores) > limit {
				return scores[:limit], nil
			}
			return scores, nil
		}
	}

	// Calculate trending scores
	filter := bson.M{
		"privacy":    models.EventPrivacyPublic,
		"start_date": bson.M{"$gt": time.Now()},
	}
	events, _, err := s.eventRepo.List(ctx, 100, 1, filter)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	scores := []TrendingScore{}

	for _, event := range events {
		// Calculate trending score
		// Formula: (going * 2) + (interested * 1) + (shares * 3) - (age_in_hours * 0.1)
		hoursOld := now.Sub(event.CreatedAt).Hours()
		score := float64(event.Stats.GoingCount)*2.0 +
			float64(event.Stats.InterestedCount)*1.0 +
			float64(event.Stats.ShareCount)*3.0 -
			hoursOld*0.1

		if score > 0 {
			e := event
			scores = append(scores, TrendingScore{
				EventID: event.ID.Hex(),
				Score:   score,
				Event:   &e,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Limit and cache
	if len(scores) > limit {
		scores = scores[:limit]
	}

	// Cache the results
	if s.eventCache != nil {
		eventIDs := []string{}
		for _, s := range scores {
			eventIDs = append(eventIDs, s.EventID)
		}
		s.eventCache.SetTrendingEvents(ctx, eventIDs)
	}

	return scores, nil
}
