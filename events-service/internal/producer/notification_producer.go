package producer

import (
	"context"
	"encoding/json"
	"log"
	"gitlab.com/spydotech-group/shared-entity/models"
	"gitlab.com/spydotech-group/shared-entity/events"
	"time"

	"github.com/segmentio/kafka-go"
)

type NotificationProducer struct {
	writer *kafka.Writer
}

func NewNotificationProducer(brokers []string, topic string) *NotificationProducer {
	return &NotificationProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *NotificationProducer) PublishNotification(ctx context.Context, notification *models.Notification) error {
	event := events.NotificationCreatedEvent{
		ID:          notification.ID,
		RecipientID: notification.RecipientID,
		SenderID:    notification.SenderID,
		Type:        string(notification.Type), // Ensure cast if enum
		TargetID:    notification.TargetID,
		TargetType:  string(notification.TargetType), // Ensure cast if enum
		Content:     notification.Content,
		Data:        notification.Data,
		Read:        notification.Read,
		CreatedAt:   notification.CreatedAt,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(notification.RecipientID.Hex()), // Partition by recipient
		Value: payload,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("Failed to write notification to Kafka: %v", err)
		return err
	}
	return nil
}

func (p *NotificationProducer) Close() error {
	return p.writer.Close()
}
