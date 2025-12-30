package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/internal/idempotency"
)

func TestIdempotencyService_NilClient(t *testing.T) {
	// Service should work gracefully without Redis (dev mode)
	service := idempotency.NewService(nil, time.Hour)

	ctx := context.Background()
	isDuplicate, err := service.Check(ctx, "test-event-1")

	if err != nil {
		t.Errorf("Expected no error with nil client, got: %v", err)
	}
	if isDuplicate {
		t.Error("Expected isDuplicate=false with nil client")
	}
}

func TestIdempotencyService_MarkProcessed(t *testing.T) {
	// Service should work gracefully without Redis
	service := idempotency.NewService(nil, time.Hour)

	ctx := context.Background()
	err := service.MarkProcessed(ctx, "test-event-2")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
