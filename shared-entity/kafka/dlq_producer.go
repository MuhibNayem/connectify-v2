package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type DLQProducer struct {
	writers map[string]*kafka.Writer
	brokers []string
}

func NewDLQProducer(brokers []string) *DLQProducer {
	return &DLQProducer{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

func (p *DLQProducer) PublishDeadLetter(ctx context.Context, originalTopic string, message []byte, reason error) error {
	dlqTopic := originalTopic + ".dlq"

	writer, exists := p.writers[dlqTopic]
	if !exists {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(p.brokers...),
			Topic:    dlqTopic,
			Balancer: &kafka.LeastBytes{},
		}
		p.writers[dlqTopic] = writer
	}

	// Wrap message with metadata
	dlqMessage := map[string]interface{}{
		"original_topic": originalTopic,
		"payload":        string(message),
		"error":          reason.Error(),
		"timestamp":      time.Now(),
	}

	payload, err := json.Marshal(dlqMessage)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Value: payload,
		Time:  time.Now(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("CRITICAL: Failed to write to DLQ %s: %v", dlqTopic, err)
		return err
	}

	log.Printf("Successfully sent failed message to DLQ: %s", dlqTopic)
	return nil
}

func (p *DLQProducer) Close() {
	for _, w := range p.writers {
		w.Close()
	}
}
