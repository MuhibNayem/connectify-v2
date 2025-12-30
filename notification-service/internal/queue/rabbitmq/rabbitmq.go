package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitQueue implements a robust RabbitMQ adapter with auto-reconnection and publisher confirms.
type RabbitQueue struct {
	url       string
	topic     string
	conn      *amqp.Connection
	ch        *amqp.Channel
	mu        sync.RWMutex // Protects conn/ch access
	logger    *zap.Logger  // Assuming we can inject logger, if not will use standard log or fmt/error
	closed    bool
	done      chan struct{}
	notifyCos chan *amqp.Error // Connection close notification
	notifyCh  chan *amqp.Error // Channel close notification
}

// NewRabbitQueue initializes a new RabbitMQ adapter.
func NewRabbitQueue(url, topic string) (*RabbitQueue, error) {
	rq := &RabbitQueue{
		url:   url,
		topic: topic,
		done:  make(chan struct{}),
	}

	if err := rq.connect(); err != nil {
		return nil, err
	}

	// Start background reconnection monitor
	go rq.reconnectLoop()

	return rq, nil
}

// connect establishes the connection and channel, declares topology, and enables confirms
func (r *RabbitQueue) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		return fmt.Errorf("failed to dial rabbitmq: %w", err)
	}

	r.ch, err = r.conn.Channel()
	if err != nil {
		r.conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 1. Enable Publisher Confirms (Review Requirement: Data Safety)
	if err := r.ch.Confirm(false); err != nil {
		r.ch.Close()
		r.conn.Close()
		return fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	// 2. Declare Durable Queue with DLX
	// We use the topic name as the queue name.
	args := amqp.Table{
		"x-queue-type":              "quorum",
		"x-dead-letter-exchange":    "",               // Default exchange
		"x-dead-letter-routing-key": r.topic + "-dlq", // Route rejected msgs to DLQ queue
	}

	_, err = r.ch.QueueDeclare(
		r.topic, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		args,
	)
	// Fallback if quorum not supported or simple queue preferred:
	if err != nil {
		// Retry as classic durable queue if x-queue-type fails (backward compatibility)
		_, err = r.ch.QueueDeclare(
			r.topic, true, false, false, false, nil,
		)
		if err != nil {
			r.ch.Close()
			r.conn.Close()
			return fmt.Errorf("failed to declare queue: %w", err)
		}
	}

	// 3. Set QoS (Fair Dispatch / Backpressure)
	// Prefetch 10: Worker processes 10 at a time max. Prevents RAM overload.
	if err := r.ch.Qos(20, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// 4. Register Close Listeners
	r.notifyCos = make(chan *amqp.Error, 1)
	r.conn.NotifyClose(r.notifyCos)

	r.notifyCh = make(chan *amqp.Error, 1)
	r.ch.NotifyClose(r.notifyCh)

	return nil
}

func (r *RabbitQueue) reconnectLoop() {
	for {
		select {
		case <-r.done:
			return
		case err := <-r.notifyCos:
			if err != nil {
				r.handleReconnect("connection", err)
			}
		case err := <-r.notifyCh:
			if err != nil {
				r.handleReconnect("channel", err)
			}
		}
	}
}

func (r *RabbitQueue) handleReconnect(typ string, cause error) {
	fmt.Printf("RabbitMQ %s closed: %v. Reconnecting...\n", typ, cause)

	for {
		// Exponential Backoff could be added here
		time.Sleep(2 * time.Second)

		if err := r.connect(); err == nil {
			fmt.Println("RabbitMQ reconnected")
			return
		} else {
			fmt.Printf("Failed to reconnect: %v. Retrying...\n", err)
		}
	}
}

// Publish sends a message and waits for confirmation (Sync)
func (r *RabbitQueue) Publish(ctx context.Context, event *adapters.NotificationEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	r.mu.RLock()
	ch := r.ch // Copy pointer
	r.mu.RUnlock()

	if ch == nil {
		return errors.New("connection closed")
	}

	// Create a confirmation listener for this publish
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	err = ch.PublishWithContext(ctx,
		"",      // exchange
		r.topic, // routing key
		true,    // mandatory: fail if no queue bound
		false,   // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Persist directly to disk
			ContentType:  "application/json",
			Body:         body,
			MessageId:    event.ID,
			Timestamp:    time.Now(),
		})

	if err != nil {
		return err
	}

	// Wait for confirmation from Broker (Data Safety)
	select {
	case confirmed := <-confirms:
		if confirmed.Ack {
			return nil
		}
		return errors.New("message nacked by broker")
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second): // Safety timeout
		return errors.New("publish confirmation timeout")
	}
}

func (r *RabbitQueue) PublishBatch(ctx context.Context, events []*adapters.NotificationEvent) error {
	for _, event := range events {
		if err := r.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (r *RabbitQueue) Subscribe(ctx context.Context, handler adapters.EventHandler) error {
	r.mu.RLock()
	ch := r.ch
	r.mu.RUnlock()

	if ch == nil {
		return errors.New("not connected")
	}

	// Unique consumer tag
	consumerTag := fmt.Sprintf("consumer-%d", time.Now().UnixNano())

	msgs, err := ch.Consume(
		r.topic,     // queue
		consumerTag, // consumer
		false,       // auto-ack (MANUAL is cleaner for reliability)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			// Handle Context Cancellation
			if ctx.Err() != nil {
				return
			}

			var event adapters.NotificationEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				// Poison Pill: Reject without requeue
				d.Nack(false, false)
				continue
			}

			// INJECT MANUAL ACK/NACK
			// This allows the specific worker to Ack AFTER it completes processing.
			// Critical for "At-Least-Once" guarantee.
			event.Ack = func() error {
				return d.Ack(false)
			}
			event.Nack = func() error {
				return d.Nack(false, true) // Requeue enabled for retries
			}

			if err := handler(ctx, &event); err != nil {
				// If handler returns error (e.g. queue full backpressure), we Nack here.
				// But generally, the handler (Orchestrator) takes ownership.
				d.Nack(false, true)
			}
			// NO AUTO ACK HERE! Handler (Orchestrator) must call event.Ack()
		}

	}()

	return nil
}

func (r *RabbitQueue) Close() error {
	close(r.done)
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	return nil
}

func (r *RabbitQueue) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.conn.IsClosed() {
		return errors.New("connection closed")
	}
	return nil
}
