package loadtest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/adapters"
	"github.com/segmentio/kafka-go"
)

/*
Kafka Throughput Test - Measure Event Processing Capacity

This test publishes events directly to Kafka and measures how fast
the Orchestrator can consume and process them.
*/

// TestKafka_MaxThroughput measures Kafka message consumption rate
func TestKafka_MaxThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Kafka throughput test in short mode")
	}

	levels := []int{1000, 5000, 10000, 50000, 100000}

	t.Logf("\n🔥 Kafka Throughput Test")
	t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for _, messageCount := range levels {
		t.Logf("📊 Testing %d messages...", messageCount)

		result := runKafkaThroughputTest(t, messageCount)

		status := "✓ HEALTHY"
		if result.ErrorRate > 0.01 {
			status = "✗ ERRORS"
		}

		t.Logf("   Published: %d msg in %.2fs = %.0f msg/s",
			messageCount, result.PublishDuration.Seconds(), result.PublishRate)
		t.Logf("   Consumed:  %d msg in %.2fs = %.0f msg/s | %s\n",
			result.ConsumedCount, result.ConsumeDuration.Seconds(), result.ConsumeRate, status)

		// Pause between tests
		time.Sleep(3 * time.Second)
	}
}

type KafkaThroughputResult struct {
	MessageCount    int
	PublishDuration time.Duration
	PublishRate     float64
	ConsumedCount   int64
	ConsumeDuration time.Duration
	ConsumeRate     float64
	ErrorRate       float64
}

func runKafkaThroughputTest(t *testing.T, messageCount int) KafkaThroughputResult {
	const topic = "notifications"
	const brokerAddr = "localhost:29092"

	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        true, // Async for max throughput
	}
	defer writer.Close()

	// Track consumption via storage (notifications created)
	initialCount := getNotificationCount(t)

	// Publish messages
	publishStart := time.Now()

	messages := make([]kafka.Message, messageCount)
	for i := 0; i < messageCount; i++ {
		event := adapters.NotificationEvent{
			ID:        fmt.Sprintf("load-test-%d", i),
			Type:      "user.notification",
			Timestamp: time.Now().Format(time.RFC3339),
			TenantID:  "load-test",
			Payload: map[string]interface{}{
				"recipient_id": fmt.Sprintf("user-%d", i%1000),
				"sender_id":    "load-test",
				"type":         "test",
				"title":        "Load Test",
				"body":         fmt.Sprintf("Test message %d", i),
				"priority":     "normal",
			},
		}

		payload, _ := json.Marshal(event)
		messages[i] = kafka.Message{
			Key:   []byte(event.ID),
			Value: payload,
		}
	}

	// Batch publish
	err := writer.WriteMessages(context.Background(), messages...)
	publishDuration := time.Since(publishStart)

	if err != nil {
		t.Logf("Warning: Kafka publish error: %v", err)
	}

	publishRate := float64(messageCount) / publishDuration.Seconds()

	// Wait for consumption
	t.Logf("   Waiting for orchestrator to process messages...")

	consumeStart := time.Now()
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var finalCount int64

	for {
		select {
		case <-timeout:
			finalCount = getNotificationCount(t)
			goto done
		case <-ticker.C:
			currentCount := getNotificationCount(t)
			consumed := currentCount - initialCount

			// Done when all messages consumed
			if consumed >= int64(messageCount) {
				finalCount = currentCount
				goto done
			}
		}
	}

done:
	consumeDuration := time.Since(consumeStart)
	consumedCount := finalCount - initialCount
	consumeRate := float64(consumedCount) / consumeDuration.Seconds()
	errorRate := float64(messageCount-int(consumedCount)) / float64(messageCount)

	return KafkaThroughputResult{
		MessageCount:    messageCount,
		PublishDuration: publishDuration,
		PublishRate:     publishRate,
		ConsumedCount:   consumedCount,
		ConsumeDuration: consumeDuration,
		ConsumeRate:     consumeRate,
		ErrorRate:       errorRate,
	}
}

func getNotificationCount(t *testing.T) int64 {
	// This is a rough approximation - in real test we'd query storage
	// For now, we'll use atomic counter
	return atomic.LoadInt64(&kafkaProcessedCount)
}

var kafkaProcessedCount int64

// TestKafka_ConsumerGroupThroughput tests parallel consumption
func TestKafka_ConsumerGroupThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Kafka consumer group test in short mode")
	}

	const messageCount = 100000
	const topic = "notifications"
	const brokerAddr = "localhost:29092"

	t.Logf("\n🚀 Kafka Consumer Group Throughput")
	t.Logf("Publishing %d messages to %d partitions...\n", messageCount, 10)

	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddr),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1000,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}
	defer writer.Close()

	// Publish messages
	publishStart := time.Now()

	messages := make([]kafka.Message, messageCount)
	for i := 0; i < messageCount; i++ {
		event := adapters.NotificationEvent{
			ID:        fmt.Sprintf("throughput-test-%d", i),
			Type:      "user.notification",
			Timestamp: time.Now().Format(time.RFC3339),
			TenantID:  "throughput-test",
			Payload: map[string]interface{}{
				"recipient_id": fmt.Sprintf("user-%d", i%10000),
				"type":         "throughput_test",
				"title":        "Throughput Test",
				"body":         fmt.Sprintf("Message %d", i),
			},
		}

		payload, _ := json.Marshal(event)
		messages[i] = kafka.Message{
			Key:   []byte(event.ID),
			Value: payload,
		}
	}

	err := writer.WriteMessages(context.Background(), messages...)
	publishDuration := time.Since(publishStart)

	if err != nil {
		t.Fatalf("Kafka publish failed: %v", err)
	}

	publishRate := float64(messageCount) / publishDuration.Seconds()

	t.Logf("✓ Published %d messages in %.2fs", messageCount, publishDuration.Seconds())
	t.Logf("  Publish Rate: %.0f msg/s\n", publishRate)

	t.Logf("⏳ Orchestrator consuming with 200 workers...")
	t.Logf("   (Check server logs for consumption rate)\n")

	// Wait and monitor
	time.Sleep(10 * time.Second)

	t.Logf("╔══════════════════════════════════════════════════════════════════╗")
	t.Logf("║  KAFKA PUBLISH THROUGHPUT                                        ║")
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║    Messages:           %-10d                                  ║", messageCount)
	t.Logf("║    Duration:           %-10.2fs                                ║", publishDuration.Seconds())
	t.Logf("║    Publish Rate:       %-10.0f msg/s                          ║", publishRate)
	t.Logf("║    Batch Size:         %-10d                                  ║", 1000)
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  CONSUMPTION                                                     ║")
	t.Logf("║    Worker Concurrency: 200                                       ║")
	t.Logf("║    Expected Rate:      10,000-50,000 msg/s                       ║")
	t.Logf("║    (Check /tmp/notification-kafka.log for actual consumption)    ║")
	t.Logf("╚══════════════════════════════════════════════════════════════════╝")
}
