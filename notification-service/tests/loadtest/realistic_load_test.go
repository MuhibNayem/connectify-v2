package loadtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/MuhibNayem/connectify-v2/notification-service/proto/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestRealisticLoad tests with PACED requests (realistic user behavior)
func TestRealisticLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping realistic load test in short mode")
	}

	config := RealisticLoadConfig{
		ConcurrentUsers:  500,                    // 500 concurrent users
		RequestsPerUser:  100,                    // 100 requests each
		RequestPacing:    100 * time.Millisecond, // 10 req/sec per user
		Duration:         30 * time.Second,
		CreatePercentage: 30,
		ListPercentage:   70,
	}

	result := runRealisticLoadTest(config)

	t.Logf("\n╔══════════════════════════════════════════════════════════════════╗")
	t.Logf("║              REALISTIC LOAD TEST RESULTS                         ║")
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  Concurrent Users:     %-5d                                     ║", config.ConcurrentUsers)
	t.Logf("║  Request Pacing:       %v/user (realistic)                  ║", config.RequestPacing)
	t.Logf("║  Duration:             %v                                       ║", config.Duration)
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  RESULTS                                                         ║")
	t.Logf("║    Total Requests:     %-10d                                  ║", result.TotalRequests)
	t.Logf("║    Successful:         %-10d (%.2f%%)                        ║", result.SuccessRequests, result.SuccessRate*100)
	t.Logf("║    Failed:             %-10d (%.2f%%)                        ║", result.FailedRequests, result.ErrorRate*100)
	t.Logf("║    Actual RPS:         %-10.0f                                ║", result.ActualRPS)
	t.Logf("║    Expected RPS:       %-10.0f                                ║", float64(config.ConcurrentUsers)/config.RequestPacing.Seconds())
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  LATENCY                                                         ║")
	t.Logf("║    Mean:               %-10v                                  ║", result.MeanLatency)
	t.Logf("║    P50:                %-10v                                  ║", result.P50Latency)
	t.Logf("║    P95:                %-10v                                  ║", result.P95Latency)
	t.Logf("║    P99:                %-10v                                  ║", result.P99Latency)
	t.Logf("║    Max:                %-10v                                  ║", result.MaxLatency)
	t.Logf("╚══════════════════════════════════════════════════════════════════╝")

	// Assertions
	if result.ErrorRate > 0.01 {
		t.Errorf("Error rate too high: %.2f%% (max 1%%)", result.ErrorRate*100)
	}

	if result.P99Latency > 200*time.Millisecond {
		t.Logf("Warning: P99 latency high: %v (target < 200ms)", result.P99Latency)
	}

	expectedRPS := float64(config.ConcurrentUsers) / config.RequestPacing.Seconds()
	if result.ActualRPS < expectedRPS*0.9 {
		t.Logf("Warning: RPS below expected (%.0f), got %.0f", expectedRPS, result.ActualRPS)
	}
}

type RealisticLoadConfig struct {
	ConcurrentUsers  int
	RequestsPerUser  int
	RequestPacing    time.Duration
	Duration         time.Duration
	CreatePercentage int
	ListPercentage   int
}

type RealisticLoadResult struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	SuccessRate     float64
	ErrorRate       float64
	ActualRPS       float64
	MeanLatency     time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MaxLatency      time.Duration
	Duration        time.Duration
}

func runRealisticLoadTest(config RealisticLoadConfig) RealisticLoadResult {
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+5*time.Second)
	defer cancel()

	histogram := NewLatencyHistogram()
	var totalRequests, successRequests, failedRequests int64
	var maxLatency time.Duration

	// Create gRPC clients (one per user)
	clients := make([]pb.NotificationServiceClient, min(config.ConcurrentUsers, 100))
	for i := range clients {
		conn, err := grpc.Dial(grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024)),
		)
		if err != nil {
			continue
		}
		defer conn.Close()
		clients[i] = pb.NewNotificationServiceClient(conn)
	}

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch users with PACING
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			clientIdx := userID % len(clients)
			if clients[clientIdx] == nil {
				return
			}

			ticker := time.NewTicker(config.RequestPacing)
			defer ticker.Stop()

			requestCount := 0

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if requestCount >= config.RequestsPerUser {
						return
					}

					reqStart := time.Now()

					// Determine operation
					var err error
					opType := int(atomic.AddInt64(&totalRequests, 1)) % 100
					if opType < config.CreatePercentage {
						err = createNotificationGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
					} else {
						err = listNotificationsGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
					}

					latency := time.Since(reqStart)
					histogram.Record(latency)

					if err == nil {
						atomic.AddInt64(&successRequests, 1)
					} else {
						atomic.AddInt64(&failedRequests, 1)
					}

					if latency > maxLatency {
						maxLatency = latency
					}

					requestCount++
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	total := atomic.LoadInt64(&totalRequests)
	success := atomic.LoadInt64(&successRequests)
	failed := atomic.LoadInt64(&failedRequests)

	return RealisticLoadResult{
		TotalRequests:   total,
		SuccessRequests: success,
		FailedRequests:  failed,
		SuccessRate:     float64(success) / float64(total),
		ErrorRate:       float64(failed) / float64(total),
		ActualRPS:       float64(total) / duration.Seconds(),
		MeanLatency:     histogram.Mean(),
		P50Latency:      histogram.Percentile(0.50),
		P95Latency:      histogram.Percentile(0.95),
		P99Latency:      histogram.Percentile(0.99),
		MaxLatency:      maxLatency,
		Duration:        duration,
	}
}
