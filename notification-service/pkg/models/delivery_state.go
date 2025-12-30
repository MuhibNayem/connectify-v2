package models

import "time"

// DeliveryStatus represents the state machine for a single channel delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSent      DeliveryStatus = "sent"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

// ChannelDelivery tracks delivery state per channel
type ChannelDelivery struct {
	Channel      string         `json:"channel" bson:"channel"`
	Status       DeliveryStatus `json:"status" bson:"status"`
	Attempts     int            `json:"attempts" bson:"attempts"`
	LastAttempt  *time.Time     `json:"last_attempt,omitempty" bson:"last_attempt,omitempty"`
	DeliveredAt  *time.Time     `json:"delivered_at,omitempty" bson:"delivered_at,omitempty"`
	FailedAt     *time.Time     `json:"failed_at,omitempty" bson:"failed_at,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty" bson:"error_message,omitempty"`
	Recipient    string         `json:"recipient,omitempty" bson:"recipient,omitempty"` // email/phone/token used
}

// NotificationDeliveryState holds delivery state for all channels
type NotificationDeliveryState struct {
	NotificationID string             `json:"notification_id" bson:"notification_id"`
	Channels       []*ChannelDelivery `json:"channels" bson:"channels"`
	OverallStatus  DeliveryStatus     `json:"overall_status" bson:"overall_status"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}

// NewDeliveryState creates initial delivery state for a notification
func NewDeliveryState(notificationID string, channels []string) *NotificationDeliveryState {
	now := time.Now()
	deliveries := make([]*ChannelDelivery, len(channels))
	for i, ch := range channels {
		deliveries[i] = &ChannelDelivery{
			Channel:  ch,
			Status:   DeliveryStatusPending,
			Attempts: 0,
		}
	}
	return &NotificationDeliveryState{
		NotificationID: notificationID,
		Channels:       deliveries,
		OverallStatus:  DeliveryStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateChannelStatus updates delivery status for a specific channel
func (s *NotificationDeliveryState) UpdateChannelStatus(channel string, status DeliveryStatus, errMsg string) {
	now := time.Now()
	for _, cd := range s.Channels {
		if cd.Channel == channel {
			cd.Status = status
			cd.Attempts++
			cd.LastAttempt = &now
			if status == DeliveryStatusDelivered {
				cd.DeliveredAt = &now
			} else if status == DeliveryStatusFailed {
				cd.FailedAt = &now
				cd.ErrorMessage = errMsg
			}
			break
		}
	}
	s.UpdatedAt = now
	s.recalculateOverallStatus()
}

// recalculateOverallStatus determines overall status based on channel statuses
func (s *NotificationDeliveryState) recalculateOverallStatus() {
	allDelivered := true
	anyFailed := false
	anyPending := false

	for _, cd := range s.Channels {
		switch cd.Status {
		case DeliveryStatusDelivered, DeliveryStatusSent:
			// Good
		case DeliveryStatusFailed:
			anyFailed = true
			allDelivered = false
		case DeliveryStatusPending, DeliveryStatusRetrying:
			anyPending = true
			allDelivered = false
		}
	}

	if allDelivered {
		s.OverallStatus = DeliveryStatusDelivered
	} else if anyPending {
		s.OverallStatus = DeliveryStatusPending
	} else if anyFailed {
		s.OverallStatus = DeliveryStatusFailed
	}
}

// GetChannelStatus returns status for a specific channel
func (s *NotificationDeliveryState) GetChannelStatus(channel string) *ChannelDelivery {
	for _, cd := range s.Channels {
		if cd.Channel == channel {
			return cd
		}
	}
	return nil
}
