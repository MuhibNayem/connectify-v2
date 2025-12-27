package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/events-service/internal/integration"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/pkg/async"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/service/mocks"
	"github.com/MuhibNayem/connectify-v2/events-service/internal/service/testutil"
	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestEventService_CreateEvent tests event creation functionality
func TestEventService_CreateEvent(t *testing.T) {
	userID := primitive.NewObjectID()
	tests := []struct {
		name          string
		userID        primitive.ObjectID
		req           *models.CreateEventRequest
		mockSetup     func(*mocks.MockEventRepository, *mocks.MockEventBroadcaster)
		wantErr       bool
		errContains   string
		validateEvent func(*testing.T, *models.Event)
	}{
		{
			name:   "successful event creation",
			userID: userID,
			req:    testutil.NewCreateEventRequestBuilder().WithTitle("Team Meetup").Build(),
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.CreateFunc = func(ctx context.Context, event *models.Event) error {
					event.ID = primitive.NewObjectID()
					return nil
				}
			},
			wantErr: false,
			validateEvent: func(t *testing.T, event *models.Event) {
				if event == nil {
					t.Fatal("expected event to be created, got nil")
				}
				// Verify event created with correct data
				assert.NotNil(t, event.ID)
				assert.Equal(t, "Team Meetup", event.Title)
				assert.Equal(t, "Test Description", event.Description)
				assert.Equal(t, userID, event.CreatorID)
				assert.Equal(t, models.EventPrivacyPublic, event.Privacy)
				assert.Equal(t, int64(1), event.Stats.GoingCount)

				// Verify creator is attendee
				assert.Len(t, event.Attendees, 1)
				assert.Equal(t, userID, event.Attendees[0].UserID)
				assert.Equal(t, models.RSVPStatusGoing, event.Attendees[0].Status)
				assert.WithinDuration(t, time.Now(), event.Attendees[0].Timestamp, time.Second)

				// Verify repo call - This check is already outside the validateEvent func, removing duplicate
			},
		},
		{
			name:   "event creation with repository error",
			userID: primitive.NewObjectID(),
			req:    testutil.NewCreateEventRequestBuilder().Build(),
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.CreateFunc = func(ctx context.Context, event *models.Event) error {
					return errors.New("database connection failed")
				}
			},
			wantErr:     true,
			errContains: "database connection failed",
		},
		{
			name:   "event creation with private privacy",
			userID: primitive.NewObjectID(),
			req:    testutil.NewCreateEventRequestBuilder().WithPrivacy(models.EventPrivacyPrivate).Build(),
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.CreateFunc = func(ctx context.Context, event *models.Event) error {
					event.ID = primitive.NewObjectID()
					return nil
				}
			},
			wantErr: false,
			validateEvent: func(t *testing.T, event *models.Event) {
				if event.Privacy != models.EventPrivacyPrivate {
					t.Errorf("expected privacy private, got '%s'", event.Privacy)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockRepo := &mocks.MockEventRepository{}
			mockBroadcaster := &mocks.MockEventBroadcaster{}
			tt.mockSetup(mockRepo, mockBroadcaster)

			// Create service with mocks
			svc := &EventService{
				eventRepo:   mockRepo,
				userRepo:    nil, // Not needed for this test
				broadcaster: mockBroadcaster,
				asyncRunner: async.NewRunner(slog.Default()),
			}

			// Execute
			ctx := context.Background()
			event, err := svc.CreateEvent(ctx, tt.userID, *tt.req)

			// Verify error
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate result
			if tt.validateEvent != nil {
				tt.validateEvent(t, event)
			}

			// Verify repository was called
			if mockRepo.CreateCalls != 1 {
				t.Errorf("expected Create to be called once, got %d calls", mockRepo.CreateCalls)
			}
		})
	}
}

// TestEventService_GetEvent tests retrieving a single event
func TestEventService_GetEvent(t *testing.T) {
	eventID := primitive.NewObjectID()
	viewerID := primitive.NewObjectID()

	tests := []struct {
		name        string
		eventID     primitive.ObjectID
		viewerID    primitive.ObjectID
		mockSetup   func(*mocks.MockEventRepository, *mocks.MockUserRepo)
		wantErr     bool
		validateRes func(*testing.T, *models.EventResponse)
	}{
		{
			name:     "successfully get public event",
			eventID:  eventID,
			viewerID: viewerID,
			mockSetup: func(repo *mocks.MockEventRepository, userRepo *mocks.MockUserRepo) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return testutil.NewEventBuilder().
						WithID(eventID).
						WithTitle("Tech Conference 2025").
						WithPrivacy(models.EventPrivacyPublic).
						Build(), nil
				}
				userRepo.FindByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error) {
					return &integration.EventUser{Username: "host_user"}, nil
				}
			},
			wantErr: false,
			validateRes: func(t *testing.T, res *models.EventResponse) {
				if res.Title != "Tech Conference 2025" {
					t.Errorf("expected title 'Tech Conference 2025', got '%s'", res.Title)
				}
			},
		},
		{
			name:     "event not found",
			eventID:  eventID,
			viewerID: viewerID,
			mockSetup: func(repo *mocks.MockEventRepository, userRepo *mocks.MockUserRepo) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return nil, errors.New("event not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockEventRepository{}
			mockUserRepo := &mocks.MockUserRepo{}
			tt.mockSetup(mockRepo, mockUserRepo)

			svc := &EventService{
				eventRepo:   mockRepo,
				userRepo:    mockUserRepo,
				asyncRunner: async.NewRunner(slog.Default()),
			}

			res, err := svc.GetEvent(context.Background(), tt.eventID, tt.viewerID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateRes != nil {
				tt.validateRes(t, res)
			}
		})
	}
}

// TestEventService_RSVP tests RSVP functionality
func TestEventService_RSVP(t *testing.T) {
	eventID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	tests := []struct {
		name          string
		eventID       primitive.ObjectID
		userID        primitive.ObjectID
		status        models.RSVPStatus
		mockSetup     func(*mocks.MockEventRepository, *mocks.MockEventBroadcaster)
		wantErr       bool
		validateCalls func(*testing.T, *mocks.MockEventRepository)
	}{
		{
			name:    "successful RSVP going",
			eventID: eventID,
			userID:  userID,
			status:  models.RSVPStatusGoing,
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					// Mock existing event
					existingEvent := testutil.NewEventBuilder().
						WithID(eventID).
						WithCreatorID(userID).
						WithTitle("Old Title").
						WithPrivacy(models.EventPrivacyPublic).
						Build()
					return existingEvent, nil
				}
				repo.AddOrUpdateAttendeeFunc = func(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error {
					return nil
				}
				repo.UpdateStatsFunc = func(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error {
					return nil
				}
			},
			wantErr: false,
			validateCalls: func(t *testing.T, repo *mocks.MockEventRepository) {
				if repo.AddOrUpdateAttendeeCalls != 1 {
					t.Errorf("expected AddOrUpdateAttendee to be called once, got %d", repo.AddOrUpdateAttendeeCalls)
				}
			},
		},
		{
			name:    "RSVP to non-existent event",
			eventID: eventID,
			userID:  userID,
			status:  models.RSVPStatusGoing,
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return nil, errors.New("event not found")
				}
			},
			wantErr: true,
		},
		{
			name:    "change RSVP status from interested to going",
			eventID: eventID,
			userID:  userID,
			status:  models.RSVPStatusGoing,
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return testutil.NewEventBuilder().
						WithID(eventID).
						WithAttendee(userID, models.RSVPStatusInterested).
						Build(), nil
				}
				repo.AddOrUpdateAttendeeFunc = func(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error {
					return nil
				}
				repo.UpdateStatsFunc = func(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error {
					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockEventRepository{}
			mockBroadcaster := &mocks.MockEventBroadcaster{}
			tt.mockSetup(mockRepo, mockBroadcaster)

			svc := &EventService{
				eventRepo:   mockRepo,
				broadcaster: mockBroadcaster,
				asyncRunner: async.NewRunner(slog.Default()),
			}

			err := svc.RSVP(context.Background(), tt.eventID, tt.userID, tt.status)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateCalls != nil {
				tt.validateCalls(t, mockRepo)
			}
		})
	}
}

// TestEventService_UpdateEvent tests event update functionality
func TestEventService_UpdateEvent(t *testing.T) {
	eventID := primitive.NewObjectID()
	hostID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	tests := []struct {
		name      string
		eventID   primitive.ObjectID
		userID    primitive.ObjectID
		req       models.UpdateEventRequest
		mockSetup func(*mocks.MockEventRepository, *mocks.MockUserRepo)
		wantErr   bool
	}{
		{
			name:    "host successfully updates event",
			eventID: eventID,
			userID:  hostID,
			req: models.UpdateEventRequest{
				Title:       "Updated Event Title",
				Description: "Updated description",
			},
			mockSetup: func(repo *mocks.MockEventRepository, userRepo *mocks.MockUserRepo) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return testutil.NewEventBuilder().
						WithID(eventID).
						WithCreatorID(hostID).
						Build(), nil
				}
				repo.UpdateFunc = func(ctx context.Context, event *models.Event) error {
					return nil
				}
				userRepo.FindByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error) {
					return &integration.EventUser{Username: "host_user"}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "non-host attempts to update event",
			eventID: eventID,
			userID:  otherUserID,
			req: models.UpdateEventRequest{
				Title: "Hacked Event",
			},
			mockSetup: func(repo *mocks.MockEventRepository, userRepo *mocks.MockUserRepo) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return testutil.NewEventBuilder().
						WithID(eventID).
						WithCreatorID(hostID).
						Build(), nil
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockEventRepository{}
			mockUserRepo := &mocks.MockUserRepo{}
			tt.mockSetup(mockRepo, mockUserRepo)

			svc := &EventService{
				eventRepo:   mockRepo,
				userRepo:    mockUserRepo,
				asyncRunner: async.NewRunner(slog.Default()),
			}

			_, err := svc.UpdateEvent(context.Background(), tt.eventID, tt.userID, tt.req)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEventService_invalidateFriendsGoing(t *testing.T) {
	eventID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	friendID := primitive.NewObjectID()
	attendeeID := primitive.NewObjectID()

	mockRepo := &mocks.MockEventRepository{
		GetByIDFunc: func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
			return &models.Event{
				ID: eventID,
				Attendees: []models.EventAttendee{
					{UserID: userID},
					{UserID: attendeeID},
				},
			}, nil
		},
	}
	mockUserRepo := &mocks.MockUserRepo{
		FindByIDFunc: func(ctx context.Context, id primitive.ObjectID) (*integration.EventUser, error) {
			return &integration.EventUser{
				ID:      userID,
				Friends: []primitive.ObjectID{friendID},
			}, nil
		},
	}
	mockCache := &mocks.MockEventCache{}

	svc := &EventService{
		eventRepo:  mockRepo,
		userRepo:   mockUserRepo,
		eventCache: mockCache,
	}

	svc.invalidateFriendsGoing(context.Background(), eventID, userID)

	expected := map[string]bool{
		userID.Hex():     true,
		friendID.Hex():   true,
		attendeeID.Hex(): true,
	}

	if len(mockCache.InvalidateFriendsGoingCalls) == 0 {
		t.Fatalf("expected invalidation calls, got none")
	}

	for _, call := range mockCache.InvalidateFriendsGoingCalls {
		if _, ok := expected[call.UserID]; ok && call.EventID == eventID.Hex() {
			delete(expected, call.UserID)
		}
	}

	if len(expected) != 0 {
		t.Fatalf("missing invalidation for users: %v", expected)
	}
}

// TestEventService_DeleteEvent tests event deletion
func TestEventService_DeleteEvent(t *testing.T) {
	eventID := primitive.NewObjectID()
	hostID := primitive.NewObjectID()

	tests := []struct {
		name      string
		eventID   primitive.ObjectID
		userID    primitive.ObjectID
		mockSetup func(*mocks.MockEventRepository, *mocks.MockEventBroadcaster)
		wantErr   bool
	}{
		{
			name:    "host successfully deletes event",
			eventID: eventID,
			userID:  hostID,
			mockSetup: func(repo *mocks.MockEventRepository, broadcaster *mocks.MockEventBroadcaster) {
				repo.GetByIDFunc = func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
					return testutil.NewEventBuilder().
						WithID(eventID).
						WithCreatorID(hostID).
						Build(), nil
				}
				repo.DeleteFunc = func(ctx context.Context, id primitive.ObjectID) error {
					return nil
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mocks.MockEventRepository{}
			mockBroadcaster := &mocks.MockEventBroadcaster{}
			tt.mockSetup(mockRepo, mockBroadcaster)

			svc := &EventService{
				eventRepo:   mockRepo,
				broadcaster: mockBroadcaster,
				asyncRunner: async.NewRunner(slog.Default()),
			}

			err := svc.DeleteEvent(context.Background(), tt.eventID, tt.userID)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				if mockRepo.DeleteCalls != 1 {
					t.Errorf("expected Delete to be called once, got %d", mockRepo.DeleteCalls)
				}
				if mockBroadcaster.PublishEventDeletedCalls != 1 {
					t.Errorf("expected PublishEventDeleted to be called once, got %d", mockBroadcaster.PublishEventDeletedCalls)
				}
			}
		})
	}
}

// Benchmark tests for performance-critical operations
func BenchmarkEventService_CreateEvent(b *testing.B) {
	mockRepo := &mocks.MockEventRepository{
		CreateFunc: func(ctx context.Context, event *models.Event) error {
			event.ID = primitive.NewObjectID()
			return nil
		},
	}

	svc := &EventService{
		eventRepo: mockRepo,
	}

	req := testutil.NewCreateEventRequestBuilder().Build()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.CreateEvent(ctx, userID, *req)
	}
}

func BenchmarkEventService_RSVP(b *testing.B) {
	eventID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	mockRepo := &mocks.MockEventRepository{
		GetByIDFunc: func(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
			return testutil.NewEventBuilder().WithID(eventID).Build(), nil
		},
		AddOrUpdateAttendeeFunc: func(ctx context.Context, eventID primitive.ObjectID, attendee models.EventAttendee) error {
			return nil
		},
		UpdateStatsFunc: func(ctx context.Context, eventID primitive.ObjectID, stats models.EventStats) error {
			return nil
		},
	}

	svc := &EventService{
		eventRepo: mockRepo,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.RSVP(ctx, eventID, userID, models.RSVPStatusGoing)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
