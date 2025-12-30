package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/core"
	memoryqueue "github.com/MuhibNayem/connectify-v2/notification-service/internal/queue/memory"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/storage/memory"
	"github.com/MuhibNayem/connectify-v2/notification-service/internal/tracing"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"go.uber.org/zap"
)

func TestOrchestrator_CreateNotification(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()
	tracer := tracing.NewTracer("test")

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
		Tracer:  tracer,
	})

	// Test
	ctx := context.Background()
	req := &models.CreateNotificationRequest{
		RecipientID: "user-123",
		SenderID:    "system",
		Type:        "TEST",
		Title:       "Test Notification",
		Body:        "This is a test",
		Channels:    []string{"inapp"},
		Priority:    models.PriorityHigh,
	}

	notif, err := orchestrator.CreateNotification(ctx, req)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if notif.ID == "" {
		t.Error("Expected notification ID to be generated")
	}
	if notif.RecipientID != "user-123" {
		t.Errorf("Expected RecipientID 'user-123', got '%s'", notif.RecipientID)
	}
	if notif.Title != "Test Notification" {
		t.Errorf("Expected Title 'Test Notification', got '%s'", notif.Title)
	}
}

func TestOrchestrator_ListNotifications(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
	})

	ctx := context.Background()

	// Create test notifications
	for i := 0; i < 5; i++ {
		orchestrator.CreateNotification(ctx, &models.CreateNotificationRequest{
			RecipientID: "user-456",
			Type:        "TEST",
			Title:       "Notification",
			Channels:    []string{"inapp"},
		})
	}

	// Test list
	result, err := orchestrator.ListNotifications(ctx, &models.ListNotificationsRequest{
		RecipientID: "user-456",
		Limit:       10,
	})

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(result.Notifications) != 5 {
		t.Errorf("Expected 5 notifications, got %d", len(result.Notifications))
	}
}

func TestOrchestrator_MarkAsRead(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
	})

	ctx := context.Background()

	// Create notification
	notif, _ := orchestrator.CreateNotification(ctx, &models.CreateNotificationRequest{
		RecipientID: "user-789",
		Type:        "TEST",
		Title:       "Unread Notification",
		Channels:    []string{"inapp"},
	})

	// Mark as read
	err := orchestrator.MarkAsRead(ctx, notif.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify
	updated, _ := orchestrator.GetNotification(ctx, notif.ID)
	if !updated.Read {
		t.Error("Expected notification to be marked as read")
	}
}

func TestOrchestrator_GetUnreadCount(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
	})

	ctx := context.Background()
	userID := "count-test-user"

	// Create 3 notifications
	for i := 0; i < 3; i++ {
		orchestrator.CreateNotification(ctx, &models.CreateNotificationRequest{
			RecipientID: userID,
			Type:        "TEST",
			Title:       "Test",
			Channels:    []string{"inapp"},
		})
	}

	// Check unread count
	count, err := orchestrator.GetUnreadCount(ctx, userID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected unread count 3, got %d", count)
	}
}

func TestOrchestrator_BatchMarkAsRead(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
	})

	ctx := context.Background()
	userID := "batch-test-user"

	// Create notifications
	var ids []string
	for i := 0; i < 5; i++ {
		notif, _ := orchestrator.CreateNotification(ctx, &models.CreateNotificationRequest{
			RecipientID: userID,
			Type:        "TEST",
			Title:       "Test",
			Channels:    []string{"inapp"},
		})
		ids = append(ids, notif.ID)
	}

	// Batch mark as read
	updated, err := orchestrator.BatchMarkAsRead(ctx, &models.BatchMarkAsReadRequest{
		RecipientID:     userID,
		NotificationIDs: ids[:3], // Mark first 3
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if updated != 3 {
		t.Errorf("Expected 3 updated, got %d", updated)
	}

	// Check remaining unread
	count, _ := orchestrator.GetUnreadCount(ctx, userID)
	if count != 2 {
		t.Errorf("Expected 2 unread remaining, got %d", count)
	}
}

func TestOrchestrator_TTL(t *testing.T) {
	// Setup
	storage := memory.NewMemoryStorage()
	queue := memoryqueue.NewMemoryQueue()
	logger, _ := zap.NewDevelopment()

	orchestrator := core.NewOrchestrator(&core.OrchestratorConfig{
		Storage: storage,
		Queue:   queue,
		Logger:  logger,
	})

	ctx := context.Background()
	ttl := 1 * time.Hour

	// Create notification with TTL
	notif, err := orchestrator.CreateNotification(ctx, &models.CreateNotificationRequest{
		RecipientID: "ttl-user",
		Type:        "EPHEMERAL",
		Title:       "Expires Soon",
		Channels:    []string{"inapp"},
		TTL:         &ttl,
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify notification was created (TTL is handled by storage layer)
	if notif.ID == "" {
		t.Error("Expected notification to be created")
	}
}
