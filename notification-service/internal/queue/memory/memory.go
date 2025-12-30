package memory

import (
	"context"
	"sync"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
)

// MemoryQueue is an in-memory implementation for development/testing
type MemoryQueue struct {
	mu       sync.RWMutex
	handlers []adapters.EventHandler
	eventCh  chan *adapters.NotificationEvent
	stopCh   chan struct{}
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		eventCh: make(chan *adapters.NotificationEvent, 10000),
		stopCh:  make(chan struct{}),
	}
}

func (m *MemoryQueue) Publish(ctx context.Context, event *adapters.NotificationEvent) error {
	select {
	case m.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MemoryQueue) PublishBatch(ctx context.Context, events []*adapters.NotificationEvent) error {
	for _, event := range events {
		if err := m.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryQueue) Subscribe(ctx context.Context, handler adapters.EventHandler, concurrency int) error {
	if concurrency <= 0 {
		concurrency = 1
	}

	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()

	for i := 0; i < concurrency; i++ {
		go m.processEvents(ctx, handler)
	}
	return nil
}

func (m *MemoryQueue) processEvents(ctx context.Context, handler adapters.EventHandler) {
	for {
		select {
		case event := <-m.eventCh:
			if event.Ack == nil {
				event.Ack = func() error { return nil }
			}
			if event.Nack == nil {
				event.Nack = func() error {
					go func(evt *adapters.NotificationEvent) {
						_ = m.Publish(context.Background(), evt)
					}(event)
					return nil
				}
			}
			_ = handler(ctx, event)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (m *MemoryQueue) Close() error {
	close(m.stopCh)
	return nil
}

func (m *MemoryQueue) HealthCheck(ctx context.Context) error {
	return nil
}
