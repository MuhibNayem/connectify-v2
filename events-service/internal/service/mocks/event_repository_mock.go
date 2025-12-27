package mocks

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockEventRepository is a mock implementation of EventRepository for testing
type MockEventRepository struct {
	CreateFunc               func(ctx context.Context, event *models.Event) error
	GetByIDFunc              func(ctx context.Context, id primitive.ObjectID) (*models.Event, error)
	UpdateFunc               func(ctx context.Context, event *models.Event) error
	DeleteFunc               func(ctx context.Context, id primitive.ObjectID) error
	ListFunc                 func(ctx context.Context, limit, page int64, filter bson.M) ([]models.Event, int64, error)
	AddOrUpdateAttendeeFunc  func(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error
	RemoveAttendeeFunc       func(ctx context.Context, eventID, userID primitive.ObjectID) error
	UpdateStatsFunc          func(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error
	GetUserEventsFunc        func(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.Event, error)
	GetAttendeesByStatusFunc func(ctx context.Context, eventID primitive.ObjectID, status models.RSVPStatus, limit, page int64) ([]models.EventAttendee, int64, error)
	GetCategoriesFunc        func(ctx context.Context) ([]models.EventCategory, error)
	IncrementShareCountFunc  func(ctx context.Context, eventID primitive.ObjectID) error
	AddCoHostFunc            func(ctx context.Context, eventID primitive.ObjectID, coHost models.EventCoHost) error
	RemoveCoHostFunc         func(ctx context.Context, eventID, userID primitive.ObjectID) error
	IsCoHostFunc             func(ctx context.Context, eventID, userID primitive.ObjectID) (bool, error)
	SearchFunc               func(ctx context.Context, query string, filter bson.M, limit, page int64) ([]models.Event, int64, error)
	GetNearbyEventsFunc      func(ctx context.Context, lat, lng, radiusKm float64, limit, page int64) ([]models.Event, int64, error)

	// Tracking calls for verification
	CreateCalls              int
	GetByIDCalls             int
	UpdateCalls              int
	DeleteCalls              int
	AddOrUpdateAttendeeCalls int
}

func (m *MockEventRepository) Create(ctx context.Context, event *models.Event) error {
	m.CreateCalls++
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, event)
	}
	return nil
}

func (m *MockEventRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
	m.GetByIDCalls++
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEventRepository) Update(ctx context.Context, event *models.Event) error {
	m.UpdateCalls++
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, event)
	}
	return nil
}

func (m *MockEventRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	m.DeleteCalls++
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockEventRepository) List(ctx context.Context, limit, page int64, filter bson.M) ([]models.Event, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, page, filter)
	}
	return []models.Event{}, 0, nil
}

func (m *MockEventRepository) AddOrUpdateAttendee(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error {
	m.AddOrUpdateAttendeeCalls++
	if m.AddOrUpdateAttendeeFunc != nil {
		return m.AddOrUpdateAttendeeFunc(ctx, eventID, attendee)
	}
	return nil
}

func (m *MockEventRepository) RemoveAttendee(ctx context.Context, eventID, userID primitive.ObjectID) error {
	if m.RemoveAttendeeFunc != nil {
		return m.RemoveAttendeeFunc(ctx, eventID, userID)
	}
	return nil
}

func (m *MockEventRepository) UpdateStats(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error {
	if m.UpdateStatsFunc != nil {
		return m.UpdateStatsFunc(ctx, eventID, stats)
	}
	return nil
}

func (m *MockEventRepository) GetUserEvents(ctx context.Context, userID primitive.ObjectID, limit, page int64) ([]models.Event, error) {
	if m.GetUserEventsFunc != nil {
		return m.GetUserEventsFunc(ctx, userID, limit, page)
	}
	return []models.Event{}, nil
}

func (m *MockEventRepository) GetAttendeesByStatus(ctx context.Context, eventID primitive.ObjectID, status models.RSVPStatus, limit, page int64) ([]models.EventAttendee, int64, error) {
	if m.GetAttendeesByStatusFunc != nil {
		return m.GetAttendeesByStatusFunc(ctx, eventID, status, limit, page)
	}
	return []models.EventAttendee{}, 0, nil
}

func (m *MockEventRepository) GetCategories(ctx context.Context) ([]models.EventCategory, error) {
	if m.GetCategoriesFunc != nil {
		return m.GetCategoriesFunc(ctx)
	}
	return []models.EventCategory{}, nil
}

func (m *MockEventRepository) IncrementShareCount(ctx context.Context, eventID primitive.ObjectID) error {
	if m.IncrementShareCountFunc != nil {
		return m.IncrementShareCountFunc(ctx, eventID)
	}
	return nil
}

func (m *MockEventRepository) AddCoHost(ctx context.Context, eventID primitive.ObjectID, coHost models.EventCoHost) error {
	if m.AddCoHostFunc != nil {
		return m.AddCoHostFunc(ctx, eventID, coHost)
	}
	return nil
}

func (m *MockEventRepository) RemoveCoHost(ctx context.Context, eventID, userID primitive.ObjectID) error {
	if m.RemoveCoHostFunc != nil {
		return m.RemoveCoHostFunc(ctx, eventID, userID)
	}
	return nil
}

func (m *MockEventRepository) IsCoHost(ctx context.Context, eventID, userID primitive.ObjectID) (bool, error) {
	if m.IsCoHostFunc != nil {
		return m.IsCoHostFunc(ctx, eventID, userID)
	}
	return false, nil
}

func (m *MockEventRepository) Search(ctx context.Context, query string, filter bson.M, limit, page int64) ([]models.Event, int64, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filter, limit, page)
	}
	return []models.Event{}, 0, nil
}

func (m *MockEventRepository) GetNearbyEvents(ctx context.Context, lat, lng, radiusKm float64, limit, page int64) ([]models.Event, int64, error) {
	if m.GetNearbyEventsFunc != nil {
		return m.GetNearbyEventsFunc(ctx, lat, lng, radiusKm, limit, page)
	}
	return []models.Event{}, 0, nil
}
