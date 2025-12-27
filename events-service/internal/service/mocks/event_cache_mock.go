package mocks

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

// MockEventCache implements service.EventCache for testing
type MockEventCache struct {
	SetEventStatsCalls []struct {
		EventID string
		Stats   *models.EventStats
	}
	InvalidateUserRSVPStatusCalls []struct {
		UserID  string
		EventID string
	}
	SetUserRSVPStatusCalls []struct {
		UserID  string
		EventID string
		Status  models.RSVPStatus
	}
	InvalidateFriendsGoingCalls []struct {
		UserID  string
		EventID string
	}
	GetCategoriesFunc func(ctx context.Context) ([]models.EventCategory, error)
	SetCategoriesFunc func(ctx context.Context, categories []models.EventCategory) error
}

func (m *MockEventCache) SetEventStats(ctx context.Context, eventID string, stats *models.EventStats) error {
	m.SetEventStatsCalls = append(m.SetEventStatsCalls, struct {
		EventID string
		Stats   *models.EventStats
	}{EventID: eventID, Stats: stats})
	return nil
}

func (m *MockEventCache) InvalidateUserRSVPStatus(ctx context.Context, userID, eventID string) error {
	m.InvalidateUserRSVPStatusCalls = append(m.InvalidateUserRSVPStatusCalls, struct {
		UserID  string
		EventID string
	}{UserID: userID, EventID: eventID})
	return nil
}

func (m *MockEventCache) SetUserRSVPStatus(ctx context.Context, userID, eventID string, status models.RSVPStatus) error {
	m.SetUserRSVPStatusCalls = append(m.SetUserRSVPStatusCalls, struct {
		UserID  string
		EventID string
		Status  models.RSVPStatus
	}{UserID: userID, EventID: eventID, Status: status})
	return nil
}

func (m *MockEventCache) InvalidateFriendsGoing(ctx context.Context, userID, eventID string) error {
	m.InvalidateFriendsGoingCalls = append(m.InvalidateFriendsGoingCalls, struct {
		UserID  string
		EventID string
	}{UserID: userID, EventID: eventID})
	return nil
}

func (m *MockEventCache) GetCategories(ctx context.Context) ([]models.EventCategory, error) {
	if m.GetCategoriesFunc != nil {
		return m.GetCategoriesFunc(ctx)
	}
	return nil, nil
}

func (m *MockEventCache) SetCategories(ctx context.Context, categories []models.EventCategory) error {
	if m.SetCategoriesFunc != nil {
		return m.SetCategoriesFunc(ctx, categories)
	}
	return nil
}
