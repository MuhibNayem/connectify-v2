package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/MuhibNayem/connectify-v2/notification-service/proto/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	ConcurrentUsers int
	RequestsPerUser int
	Duration        time.Duration
	RampUpDuration  time.Duration
	TargetRPS       int
}

// LoadTestResult holds the results of a load test
type LoadTestResult struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	TotalDuration   time.Duration
	AvgLatency      time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MaxLatency      time.Duration
	MinLatency      time.Duration
	RPS             float64
	Latencies       []time.Duration
}

const (
	httpBaseURL = "http://localhost:8080"
	grpcAddr    = "localhost:50060"
)

// TestLoadTest_HTTP_CreateNotification tests HTTP endpoint under load
func TestLoadTest_HTTP_CreateNotification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentUsers: 100,
		RequestsPerUser: 100,
		Duration:        10 * time.Second,
	}

	result := runHTTPCreateLoadTest(config)

	t.Logf("HTTP Create Notification Load Test Results:")
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Successful: %d (%.2f%%)", result.SuccessRequests, float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Failed: %d", result.FailedRequests)
	t.Logf("  Duration: %v", result.TotalDuration)
	t.Logf("  RPS: %.2f", result.RPS)
	t.Logf("  Avg Latency: %v", result.AvgLatency)
	t.Logf("  P99 Latency: %v", result.P99Latency)
	t.Logf("  Max Latency: %v", result.MaxLatency)

	if result.RPS < 1000 {
		t.Logf("Warning: RPS below target (1000), got %.2f", result.RPS)
	}

	if result.P99Latency > 100*time.Millisecond {
		t.Logf("Warning: P99 latency above 100ms: %v", result.P99Latency)
	}
}

// TestLoadTest_gRPC_CreateNotification tests gRPC endpoint under load
func TestLoadTest_gRPC_CreateNotification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentUsers: 100,
		RequestsPerUser: 100,
		Duration:        10 * time.Second,
	}

	result := runGRPCCreateLoadTest(config)

	t.Logf("gRPC Create Notification Load Test Results:")
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Successful: %d (%.2f%%)", result.SuccessRequests, float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Failed: %d", result.FailedRequests)
	t.Logf("  Duration: %v", result.TotalDuration)
	t.Logf("  RPS: %.2f", result.RPS)
	t.Logf("  Avg Latency: %v", result.AvgLatency)
	t.Logf("  P99 Latency: %v", result.P99Latency)
	t.Logf("  Max Latency: %v", result.MaxLatency)

	if result.RPS < 2000 {
		t.Logf("Warning: gRPC RPS below target (2000), got %.2f", result.RPS)
	}
}

// TestLoadTest_gRPC_StreamNotifications tests streaming endpoint under load
func TestLoadTest_gRPC_StreamNotifications(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentUsers: 50,
		RequestsPerUser: 1000, // 1000 notifications per stream
		Duration:        15 * time.Second,
	}

	result := runGRPCStreamLoadTest(config)

	t.Logf("gRPC Stream Notifications Load Test Results:")
	t.Logf("  Total Notifications: %d", result.TotalRequests)
	t.Logf("  Successful: %d (%.2f%%)", result.SuccessRequests, float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Failed: %d", result.FailedRequests)
	t.Logf("  Duration: %v", result.TotalDuration)
	t.Logf("  RPS: %.2f (Target: 50K+)", result.RPS)
	t.Logf("  Avg Latency: %v", result.AvgLatency)
	t.Logf("  P99 Latency: %v", result.P99Latency)

	if result.RPS < 10000 {
		t.Logf("Warning: Stream RPS below expected (10K+), got %.2f", result.RPS)
	}
}

// TestLoadTest_MixedWorkload tests realistic workload mix
func TestLoadTest_MixedWorkload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	var wg sync.WaitGroup
	var totalRequests, successRequests, failedRequests int64
	var totalLatency int64

	duration := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	start := time.Now()

	// Create operations (20%)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					err := createNotificationHTTP()
					latency := time.Since(reqStart).Nanoseconds()

					atomic.AddInt64(&totalRequests, 1)
					atomic.AddInt64(&totalLatency, latency)
					if err == nil {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
				}
			}
		}()
	}

	// List operations (50%)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					err := listNotificationsHTTP()
					latency := time.Since(reqStart).Nanoseconds()

					atomic.AddInt64(&totalRequests, 1)
					atomic.AddInt64(&totalLatency, latency)
					if err == nil {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
				}
			}
		}()
	}

	// Read operations (30%)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reqStart := time.Now()
					err := getUnreadCountHTTP()
					latency := time.Since(reqStart).Nanoseconds()

					atomic.AddInt64(&totalRequests, 1)
					atomic.AddInt64(&totalLatency, latency)
					if err == nil {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
				}
			}
		}()
	}

	time.Sleep(duration)
	cancel()
	wg.Wait()

	totalDur := time.Since(start)
	avgLatency := time.Duration(0)
	if totalRequests > 0 {
		avgLatency = time.Duration(totalLatency / totalRequests)
	}

	t.Logf("Mixed Workload Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d (%.2f%%)", successRequests, float64(successRequests)/float64(totalRequests)*100)
	t.Logf("  Failed: %d", failedRequests)
	t.Logf("  Duration: %v", totalDur)
	t.Logf("  RPS: %.2f", float64(totalRequests)/totalDur.Seconds())
	t.Logf("  Avg Latency: %v", avgLatency)
}

// Helper functions

func runHTTPCreateLoadTest(config LoadTestConfig) LoadTestResult {
	var totalRequests, successRequests, failedRequests int64
	var latencies []time.Duration
	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < config.RequestsPerUser; j++ {
				reqStart := time.Now()
				err := createNotificationHTTP()
				latency := time.Since(reqStart)

				atomic.AddInt64(&totalRequests, 1)
				if err == nil {
					atomic.AddInt64(&successRequests, 1)
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	return calculateResult(totalRequests, successRequests, failedRequests, duration, latencies)
}

func runGRPCCreateLoadTest(config LoadTestConfig) LoadTestResult {
	var totalRequests, successRequests, failedRequests int64
	var latencies []time.Duration
	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup

	// Create gRPC clients pool
	clients := make([]pb.NotificationServiceClient, config.ConcurrentUsers)
	for i := 0; i < config.ConcurrentUsers; i++ {
		conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return LoadTestResult{FailedRequests: 1}
		}
		defer conn.Close()
		clients[i] = pb.NewNotificationServiceClient(conn)
	}

	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			client := clients[userID]

			for j := 0; j < config.RequestsPerUser; j++ {
				reqStart := time.Now()
				err := createNotificationGRPC(client, fmt.Sprintf("user-%d", userID))
				latency := time.Since(reqStart)

				atomic.AddInt64(&totalRequests, 1)
				if err == nil {
					atomic.AddInt64(&successRequests, 1)
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	return calculateResult(totalRequests, successRequests, failedRequests, duration, latencies)
}

func runGRPCStreamLoadTest(config LoadTestConfig) LoadTestResult {
	var totalRequests, successRequests, failedRequests int64
	var latencies []time.Duration
	var mu sync.Mutex

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				atomic.AddInt64(&failedRequests, 1)
				return
			}
			defer conn.Close()

			client := pb.NewNotificationServiceClient(conn)
			stream, err := client.StreamNotifications(context.Background())
			if err != nil {
				atomic.AddInt64(&failedRequests, 1)
				return
			}

			// Send notifications
			for j := 0; j < config.RequestsPerUser; j++ {
				reqStart := time.Now()

				err := stream.Send(&pb.CreateNotificationRequest{
					RecipientId: fmt.Sprintf("user-%d", userID),
					Type:        "test",
					Title:       "Load Test",
					Body:        "Testing streaming",
					Priority:    "NORMAL",
				})

				if err == nil {
					_, recvErr := stream.Recv()
					if recvErr == nil || recvErr == io.EOF {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}

				latency := time.Since(reqStart)
				atomic.AddInt64(&totalRequests, 1)

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}

			stream.CloseSend()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	return calculateResult(totalRequests, successRequests, failedRequests, duration, latencies)
}

func createNotificationHTTP() error {
	payload := map[string]interface{}{
		"recipient_id": "test-user-123",
		"type":         "test",
		"title":        "Test Notification",
		"body":         "Load testing",
		"priority":     "NORMAL",
		"channels":     []string{"push"},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(httpBaseURL+"/api/v1/notifications", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}

func listNotificationsHTTP() error {
	resp, err := http.Get(httpBaseURL + "/api/v1/notifications?recipient_id=test-user-123&limit=20")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}

func getUnreadCountHTTP() error {
	resp, err := http.Get(httpBaseURL + "/api/v1/notifications/unread-count?recipient_id=test-user-123")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}

func createNotificationGRPC(client pb.NotificationServiceClient, recipientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CreateNotification(ctx, &pb.CreateNotificationRequest{
		RecipientId: recipientID,
		Type:        "test",
		Title:       "gRPC Test",
		Body:        "Testing gRPC",
		Priority:    "NORMAL",
	})

	return err
}

func calculateResult(total, success, failed int64, duration time.Duration, latencies []time.Duration) LoadTestResult {
	// Sort latencies for percentile calculation
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	if len(sortedLatencies) > 1 {
		for i := 0; i < len(sortedLatencies)-1; i++ {
			for j := i + 1; j < len(sortedLatencies); j++ {
				if sortedLatencies[i] > sortedLatencies[j] {
					sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
				}
			}
		}
	}

	var avgLatency, minLatency, maxLatency time.Duration
	if len(latencies) > 0 {
		var totalLatency time.Duration
		minLatency = latencies[0]
		maxLatency = latencies[0]

		for _, l := range latencies {
			totalLatency += l
			if l < minLatency {
				minLatency = l
			}
			if l > maxLatency {
				maxLatency = l
			}
		}
		avgLatency = totalLatency / time.Duration(len(latencies))
	}

	p50Latency := time.Duration(0)
	p95Latency := time.Duration(0)
	p99Latency := time.Duration(0)

	if len(sortedLatencies) > 0 {
		p50Latency = sortedLatencies[int(float64(len(sortedLatencies))*0.50)]
		p95Latency = sortedLatencies[int(float64(len(sortedLatencies))*0.95)]
		p99Latency = sortedLatencies[int(float64(len(sortedLatencies))*0.99)]
	}

	return LoadTestResult{
		TotalRequests:   total,
		SuccessRequests: success,
		FailedRequests:  failed,
		TotalDuration:   duration,
		AvgLatency:      avgLatency,
		P50Latency:      p50Latency,
		P95Latency:      p95Latency,
		P99Latency:      p99Latency,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
		RPS:             float64(total) / duration.Seconds(),
		Latencies:       latencies,
	}
}
