package testutil

import (
	"time"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventBuilder provides a fluent API for creating test events
type EventBuilder struct {
	event *models.Event
}

// NewEventBuilder creates a new event builder with sensible defaults
func NewEventBuilder() *EventBuilder {
	now := time.Now()
	return &EventBuilder{
		event: &models.Event{
			ID:          primitive.NewObjectID(),
			Title:       "Test Event",
			Description: "This is a test event",
			CreatorID:   primitive.NewObjectID(),
			Location:    "Test Location",
			Coordinates: []float64{90.4125, 23.8103}, // Lng, Lat
			StartDate:   now.Add(24 * time.Hour),
			EndDate:     now.Add(27 * time.Hour),
			Privacy:     models.EventPrivacyPublic,
			Category:    "networking",
			IsOnline:    false,
			Attendees:   []models.EventAttendee{},
			CoHosts:     []models.EventCoHost{},
			CoverImage:  "https://example.com/cover.jpg",
			Stats: models.EventStats{
				GoingCount:      0,
				InterestedCount: 0,
				ShareCount:      0,
				InvitedCount:    0,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func (b *EventBuilder) WithID(id primitive.ObjectID) *EventBuilder {
	b.event.ID = id
	return b
}

func (b *EventBuilder) WithTitle(title string) *EventBuilder {
	b.event.Title = title
	return b
}

func (b *EventBuilder) WithDescription(desc string) *EventBuilder {
	b.event.Description = desc
	return b
}

func (b *EventBuilder) WithCreatorID(creatorID primitive.ObjectID) *EventBuilder {
	b.event.CreatorID = creatorID
	return b
}

func (b *EventBuilder) WithLocation(location string, lat, lng float64) *EventBuilder {
	b.event.Location = location
	b.event.Coordinates = []float64{lng, lat}
	return b
}

func (b *EventBuilder) WithDates(start, end time.Time) *EventBuilder {
	b.event.StartDate = start
	b.event.EndDate = end
	return b
}

func (b *EventBuilder) WithPrivacy(privacy models.EventPrivacy) *EventBuilder {
	b.event.Privacy = privacy
	return b
}

func (b *EventBuilder) WithCategory(category string) *EventBuilder {
	b.event.Category = category
	return b
}

func (b *EventBuilder) WithAttendee(userID primitive.ObjectID, status models.RSVPStatus) *EventBuilder {
	b.event.Attendees = append(b.event.Attendees, models.EventAttendee{
		UserID:    userID,
		Status:    status,
		Timestamp: time.Now(),
	})
	switch status {
	case models.RSVPStatusGoing:
		b.event.Stats.GoingCount++
	case models.RSVPStatusInterested:
		b.event.Stats.InterestedCount++
	}
	return b
}

func (b *EventBuilder) WithCoHost(userID primitive.ObjectID) *EventBuilder {
	b.event.CoHosts = append(b.event.CoHosts, models.EventCoHost{
		UserID:  userID,
		AddedAt: time.Now(),
	})
	return b
}

func (b *EventBuilder) WithCoverImage(url string) *EventBuilder {
	b.event.CoverImage = url
	return b
}

func (b *EventBuilder) WithStats(going, interested, invited, shares int64) *EventBuilder {
	b.event.Stats = models.EventStats{
		GoingCount:      going,
		InterestedCount: interested,
		InvitedCount:    invited,
		ShareCount:      shares,
	}
	return b
}

func (b *EventBuilder) WithIsOnline(isOnline bool) *EventBuilder {
	b.event.IsOnline = isOnline
	return b
}

func (b *EventBuilder) Build() *models.Event {
	return b.event
}

// CreateEventRequestBuilder builds CreateEventRequest objects for testing
type CreateEventRequestBuilder struct {
	req *models.CreateEventRequest
}

func NewCreateEventRequestBuilder() *CreateEventRequestBuilder {
	now := time.Now()
	return &CreateEventRequestBuilder{
		req: &models.CreateEventRequest{
			Title:       "Test Event",
			Description: "Test Description",
			Location:    "Test Location",
			StartDate:   now.Add(24 * time.Hour),
			EndDate:     now.Add(27 * time.Hour),
			Privacy:     models.EventPrivacyPublic,
			Category:    "networking",
			CoverImage:  "https://example.com/cover.jpg",
		},
	}
}

func (b *CreateEventRequestBuilder) WithTitle(title string) *CreateEventRequestBuilder {
	b.req.Title = title
	return b
}

func (b *CreateEventRequestBuilder) WithDescription(desc string) *CreateEventRequestBuilder {
	b.req.Description = desc
	return b
}

func (b *CreateEventRequestBuilder) WithPrivacy(privacy models.EventPrivacy) *CreateEventRequestBuilder {
	b.req.Privacy = privacy
	return b
}

func (b *CreateEventRequestBuilder) WithCategory(category string) *CreateEventRequestBuilder {
	b.req.Category = category
	return b
}

func (b *CreateEventRequestBuilder) Build() *models.CreateEventRequest {
	return b.req
}
