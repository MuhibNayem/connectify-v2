package events

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
)

type EventProducer struct {
	writer *kafka.Writer
	topic  string
}

func NewEventProducer(brokers []string, topic string) *EventProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchSize:    100,
		BatchBytes:   1024 * 1024,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Compression:  compress.Snappy,
		Async:        true,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				log.Printf("Failed to deliver message: %v", err)
			}
		},
	}

	return &EventProducer{writer: w, topic: topic}
}

func (p *EventProducer) Produce(ctx context.Context, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	})
}

func (p *EventProducer) Close() error {
	return p.writer.Close()
}
