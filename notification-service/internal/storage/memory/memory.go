package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/google/uuid"
)

// MemoryStorage is an in-memory implementation of StorageAdapter for development/testing
type MemoryStorage struct {
	mu             sync.RWMutex
	notifications  map[string]*adapters.Notification
	byRecipient    map[string][]string
	deliveryStates map[string]*models.NotificationDeliveryState
	outbox         map[string]*adapters.NotificationEvent
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		notifications:  make(map[string]*adapters.Notification),
		byRecipient:    make(map[string][]string),
		deliveryStates: make(map[string]*models.NotificationDeliveryState),
		outbox:         make(map[string]*adapters.NotificationEvent),
	}
}

func (m *MemoryStorage) Create(ctx context.Context, notification *adapters.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}

	now := time.Now().Format(time.RFC3339)
	notification.CreatedAt = now
	notification.UpdatedAt = now

	m.notifications[notification.ID] = notification
	m.byRecipient[notification.RecipientID] = append(m.byRecipient[notification.RecipientID], notification.ID)

	return nil
}

func (m *MemoryStorage) Get(ctx context.Context, id string) (*adapters.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notif, exists := m.notifications[id]
	if !exists {
		return nil, fmt.Errorf("notification not found")
	}
	return notif, nil
}

func (m *MemoryStorage) List(ctx context.Context, query *adapters.ListQuery) (*adapters.ListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := m.byRecipient[query.RecipientID]
	var filtered []*adapters.Notification

	for _, id := range ids {
		notif := m.notifications[id]
		if m.matchesFilter(notif, query) {
			filtered = append(filtered, notif)
		}
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	end := len(filtered)
	if end > limit {
		end = limit
	}

	return &adapters.ListResult{
		Notifications: filtered[:end],
		Total:         int64(len(filtered)),
		HasMore:       len(filtered) > limit,
	}, nil
}

func (m *MemoryStorage) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notif, exists := m.notifications[id]
	if !exists {
		return fmt.Errorf("notification not found")
	}

	if read, ok := updates["read"].(bool); ok {
		notif.Read = read
		if read {
			now := time.Now().Format(time.RFC3339)
			notif.ReadAt = &now
		}
	}

	notif.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (m *MemoryStorage) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notif, exists := m.notifications[id]
	if !exists {
		return fmt.Errorf("notification not found")
	}

	delete(m.notifications, id)

	ids := m.byRecipient[notif.RecipientID]
	for i, nid := range ids {
		if nid == id {
			m.byRecipient[notif.RecipientID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	return nil
}

func (m *MemoryStorage) GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := int64(0)
	for _, id := range m.byRecipient[recipientID] {
		if notif := m.notifications[id]; notif != nil && !notif.Read {
			count++
		}
	}
	return count, nil
}

func (m *MemoryStorage) BatchMarkAsRead(ctx context.Context, recipientID string, ids []string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := int64(0)
	for _, id := range ids {
		if notif, exists := m.notifications[id]; exists && notif.RecipientID == recipientID {
			notif.Read = true
			now := time.Now().Format(time.RFC3339)
			notif.ReadAt = &now
			count++
		}
	}
	return count, nil
}

func (m *MemoryStorage) DeleteExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := int64(0)

	for id, notif := range m.notifications {
		if notif.ExpiresAt != nil {
			expiresAt, err := time.Parse(time.RFC3339, *notif.ExpiresAt)
			if err == nil && expiresAt.Before(now) {
				delete(m.notifications, id)
				count++
			}
		}
	}

	return count, nil
}

func (m *MemoryStorage) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) SaveDeliveryState(ctx context.Context, state *models.NotificationDeliveryState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deliveryStates[state.NotificationID] = state
	return nil
}

func (m *MemoryStorage) GetDeliveryState(ctx context.Context, notificationID string) (*models.NotificationDeliveryState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, exists := m.deliveryStates[notificationID]
	if !exists {
		return nil, nil
	}
	return state, nil
}

func (m *MemoryStorage) CreateWithOutbox(ctx context.Context, notification *adapters.Notification, event *adapters.NotificationEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Create Notification (Logic from Create)
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	now := time.Now().Format(time.RFC3339)
	notification.CreatedAt = now
	notification.UpdatedAt = now
	m.notifications[notification.ID] = notification
	m.byRecipient[notification.RecipientID] = append(m.byRecipient[notification.RecipientID], notification.ID)

	// 2. Create Outbox Event
	m.outbox[event.ID] = event

	return nil
}

func (m *MemoryStorage) CreateBatchWithOutbox(ctx context.Context, notifications []*adapters.Notification, events []*adapters.NotificationEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Format(time.RFC3339)

	// 1. Bulk Insert Notifications
	for _, n := range notifications {
		if n.ID == "" {
			n.ID = uuid.New().String()
		}
		n.CreatedAt = now
		n.UpdatedAt = now
		m.notifications[n.ID] = n
		m.byRecipient[n.RecipientID] = append(m.byRecipient[n.RecipientID], n.ID)
	}

	// 2. Bulk Insert Outbox
	for _, e := range events {
		m.outbox[e.ID] = e
	}

	return nil
}

func (m *MemoryStorage) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*adapters.NotificationEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var events []*adapters.NotificationEvent
	count := 0
	for _, event := range m.outbox {
		events = append(events, event)
		count++
		if count >= limit {
			break
		}
	}
	return events, nil
}

func (m *MemoryStorage) DeleteOutboxEvent(ctx context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.outbox, eventID)
	return nil
}

func (m *MemoryStorage) matchesFilter(notif *adapters.Notification, query *adapters.ListQuery) bool {
	if query.Read != nil && notif.Read != *query.Read {
		return false
	}
	if len(query.Types) > 0 {
		found := false
		for _, t := range query.Types {
			if notif.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
