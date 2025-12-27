package mocks

import (
	"context"

	"github.com/MuhibNayem/connectify-v2/shared-entity/models"
)

// MockEventBroadcaster is a mock implementation for testing
type MockEventBroadcaster struct {
	BroadcastRSVPFunc            func(event models.EventRSVPEvent)
	PublishEventUpdatedFunc      func(ctx context.Context, event models.EventUpdatedEvent)
	PublishEventDeletedFunc      func(ctx context.Context, event models.EventDeletedEvent)
	PublishPostCreatedFunc       func(ctx context.Context, event models.EventPostCreatedEvent)
	PublishPostReactionFunc      func(ctx context.Context, event models.EventPostReactionEvent)
	PublishInvitationUpdatedFunc func(ctx context.Context, event models.EventInvitationUpdatedEvent)
	PublishCoHostAddedFunc       func(ctx context.Context, event models.EventCoHostAddedEvent)
	PublishCoHostRemovedFunc     func(ctx context.Context, event models.EventCoHostRemovedEvent)

	// Call tracking
	BroadcastRSVPCalls       int
	PublishEventUpdatedCalls int
	PublishEventDeletedCalls int
}

func (m *MockEventBroadcaster) BroadcastRSVP(event models.EventRSVPEvent) {
	m.BroadcastRSVPCalls++
	if m.BroadcastRSVPFunc != nil {
		m.BroadcastRSVPFunc(event)
	}
}

func (m *MockEventBroadcaster) PublishEventUpdated(ctx context.Context, event models.EventUpdatedEvent) {
	m.PublishEventUpdatedCalls++
	if m.PublishEventUpdatedFunc != nil {
		m.PublishEventUpdatedFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishEventDeleted(ctx context.Context, event models.EventDeletedEvent) {
	m.PublishEventDeletedCalls++
	if m.PublishEventDeletedFunc != nil {
		m.PublishEventDeletedFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishPostCreated(ctx context.Context, event models.EventPostCreatedEvent) {
	if m.PublishPostCreatedFunc != nil {
		m.PublishPostCreatedFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishPostReaction(ctx context.Context, event models.EventPostReactionEvent) {
	if m.PublishPostReactionFunc != nil {
		m.PublishPostReactionFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishInvitationUpdated(ctx context.Context, event models.EventInvitationUpdatedEvent) {
	if m.PublishInvitationUpdatedFunc != nil {
		m.PublishInvitationUpdatedFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishCoHostAdded(ctx context.Context, event models.EventCoHostAddedEvent) {
	if m.PublishCoHostAddedFunc != nil {
		m.PublishCoHostAddedFunc(ctx, event)
	}
}

func (m *MockEventBroadcaster) PublishCoHostRemoved(ctx context.Context, event models.EventCoHostRemovedEvent) {
	if m.PublishCoHostRemovedFunc != nil {
		m.PublishCoHostRemovedFunc(ctx, event)
	}
}
