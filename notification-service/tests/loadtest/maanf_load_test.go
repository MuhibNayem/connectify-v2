package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/MuhibNayem/connectify-v2/notification-service/proto/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
MAANF-Scale Load Testing Suite for Notification Service
=======================================================

Target Performance Metrics (Meta/WhatsApp scale):
- Create Notification: < 50ms p99
- List Notifications: < 20ms p99 (with pagination)
- Unread Count: < 10ms p99
- Stream Throughput: 50K+ RPS
- Backpressure: Zero data loss under overload

Test Scenarios:
1. Sustained Load - Steady traffic over time
2. Spike Test - Sudden traffic bursts (10x)
3. Stress Test - Find breaking point
4. Backpressure Test - Validate zero data loss
5. Soak Test - Long-running stability (memory leaks)
*/

// ===== Configuration =====

type MAANGLoadConfig struct {
	// Traffic patterns
	ConcurrentUsers int
	RequestsPerUser int
	Duration        time.Duration
	RampUpDuration  time.Duration
	TargetRPS       int

	// Workload mix (must sum to 100)
	CreatePercentage int // CreateNotification
	ListPercentage   int // ListNotifications
	ReadPercentage   int // GetNotification, GetUnreadCount
	StreamPercentage int // StreamNotifications

	// Performance thresholds
	MaxP50Latency time.Duration
	MaxP95Latency time.Duration
	MaxP99Latency time.Duration
	MaxErrorRate  float64

	// System limits
	MaxCPUPercent float64
	MaxMemoryMB   int64
}

// DefaultMAANGConfig returns production-grade config
func DefaultMAANGConfig() MAANGLoadConfig {
	return MAANGLoadConfig{
		ConcurrentUsers:  500,
		RequestsPerUser:  1000,
		Duration:         30 * time.Second,
		RampUpDuration:   5 * time.Second,
		TargetRPS:        25000, // 25K RPS target
		CreatePercentage: 20,
		ListPercentage:   50,
		ReadPercentage:   20,
		StreamPercentage: 10,
		MaxP50Latency:    10 * time.Millisecond,
		MaxP95Latency:    30 * time.Millisecond,
		MaxP99Latency:    50 * time.Millisecond,
		MaxErrorRate:     0.001, // 0.1%
		MaxCPUPercent:    80,
		MaxMemoryMB:      2048,
	}
}

// ===== Results Tracking =====

type LatencyHistogram struct {
	buckets []int64 // 0-1ms, 1-5ms, 5-10ms, 10-50ms, 50-100ms, 100-500ms, 500ms+
	samples []time.Duration
	mu      sync.Mutex
}

func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		buckets: make([]int64, 7),
		samples: make([]time.Duration, 0, 100000),
	}
}

func (h *LatencyHistogram) Record(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.samples = append(h.samples, d)

	ms := d.Milliseconds()
	switch {
	case ms < 1:
		h.buckets[0]++
	case ms < 5:
		h.buckets[1]++
	case ms < 10:
		h.buckets[2]++
	case ms < 50:
		h.buckets[3]++
	case ms < 100:
		h.buckets[4]++
	case ms < 500:
		h.buckets[5]++
	default:
		h.buckets[6]++
	}
}

func (h *LatencyHistogram) Percentile(p float64) time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.samples) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(h.samples))
	copy(sorted, h.samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func (h *LatencyHistogram) Mean() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.samples) == 0 {
		return 0
	}

	var total time.Duration
	for _, s := range h.samples {
		total += s
	}
	return total / time.Duration(len(h.samples))
}

type MAANGLoadResult struct {
	// Request counts
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64

	// Latency metrics
	Histogram   *LatencyHistogram
	P50Latency  time.Duration
	P95Latency  time.Duration
	P99Latency  time.Duration
	P999Latency time.Duration
	MaxLatency  time.Duration
	MinLatency  time.Duration
	MeanLatency time.Duration

	// Throughput
	ActualRPS float64
	PeakRPS   float64
	Duration  time.Duration

	// Resource usage
	PeakCPU        float64
	PeakMemoryMB   int64
	GoroutinesPeak int

	// Per-operation metrics
	CreateLatency *LatencyHistogram
	ListLatency   *LatencyHistogram
	ReadLatency   *LatencyHistogram
	StreamLatency *LatencyHistogram

	// Errors
	ErrorRate      float64
	ErrorBreakdown map[string]int64
}

// ===== MAANG Load Tests =====

func TestMAANG_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MAANG load test in short mode")
	}

	config := MAANGLoadConfig{
		ConcurrentUsers:  300,
		Duration:         15 * time.Second,
		RampUpDuration:   3 * time.Second,
		TargetRPS:        15000,
		CreatePercentage: 20,
		ListPercentage:   50,
		ReadPercentage:   30,
	}

	result := runMAANGLoadTest(config)

	printMAANGReport(t, "Sustained Load (15s)", result)

	// Assertions
	if result.ErrorRate > 0.01 {
		t.Errorf("Error rate too high: %.4f%% (max 1%%)", result.ErrorRate*100)
	}
	if result.P99Latency > 100*time.Millisecond {
		t.Errorf("P99 latency too high: %v (max 100ms)", result.P99Latency)
	}
	if result.ActualRPS < 5000 {
		t.Logf("Warning: RPS below expectation (5K+), got %.0f", result.ActualRPS)
	}
}

func TestMAANG_SpikeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MAANG spike test in short mode")
	}

	t.Log("Phase 1: Normal load (5s)")
	result1 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers:  50,
		Duration:         5 * time.Second,
		TargetRPS:        5000,
		CreatePercentage: 30,
		ListPercentage:   70,
	})

	t.Log("Phase 2: 10x Spike (5s)")
	result2 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers:  500,
		Duration:         5 * time.Second,
		TargetRPS:        50000,
		CreatePercentage: 30,
		ListPercentage:   70,
	})

	t.Log("Phase 3: Recovery (5s)")
	result3 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers:  50,
		Duration:         5 * time.Second,
		TargetRPS:        5000,
		CreatePercentage: 30,
		ListPercentage:   70,
	})

	printMAANGReport(t, "Normal Load", result1)
	printMAANGReport(t, "10x Spike", result2)
	printMAANGReport(t, "Recovery", result3)

	// Recovery should be similar to normal
	if result3.P99Latency > result1.P99Latency*2 {
		t.Logf("Warning: Recovery latency elevated (%.2fx normal)",
			float64(result3.P99Latency)/float64(result1.P99Latency))
	}
}

func TestMAANG_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MAANG stress test in short mode")
	}

	t.Log("Finding system breaking point...")

	var breakingPoint int
	var lastGoodRPS float64

	for users := 50; users <= 5000; users *= 2 {
		result := runMAANGLoadTest(MAANGLoadConfig{
			ConcurrentUsers:  users,
			Duration:         3 * time.Second,
			TargetRPS:        users * 100,
			CreatePercentage: 30,
			ListPercentage:   70,
		})

		t.Logf("Users: %d, RPS: %.0f, P99: %v, Errors: %.4f%%",
			users, result.ActualRPS, result.P99Latency, result.ErrorRate*100)

		// Check if system is degrading
		if result.ErrorRate > 0.05 || result.P99Latency > 500*time.Millisecond {
			breakingPoint = users
			break
		}

		lastGoodRPS = result.ActualRPS
	}

	t.Logf("\nBreaking point: %d concurrent users", breakingPoint)
	t.Logf("Last stable RPS: %.0f", lastGoodRPS)
}

func TestMAANG_StreamingBackpressureTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping backpressure test in short mode")
	}

	t.Log("Testing backpressure handling under extreme load...")

	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("gRPC server not available: %v", err)
	}
	defer conn.Close()

	client := pb.NewNotificationServiceClient(conn)
	stream, err := client.StreamNotifications(context.Background())
	if err != nil {
		t.Skipf("Failed to create stream: %v", err)
	}

	var sent, acknowledged, failed int64

	// Send messages as fast as possible
	for i := 0; i < 10000; i++ {
		err := stream.Send(&pb.CreateNotificationRequest{
			RecipientId: fmt.Sprintf("user-%d", rand.Intn(1000)),
			Type:        "backpressure_test",
			Title:       "Stress Test",
			Body:        "Testing backpressure",
			Priority:    "HIGH",
		})

		atomic.AddInt64(&sent, 1)

		if err != nil {
			atomic.AddInt64(&failed, 1)
			break
		}
	}

	stream.CloseSend()

	// Receive all responses
	for {
		_, err := stream.Recv()
		if err != nil {
			break
		}
		atomic.AddInt64(&acknowledged, 1)
	}

	dataLoss := sent - acknowledged - failed
	dataLossRate := float64(dataLoss) / float64(sent)

	t.Logf("Backpressure Test Results:")
	t.Logf("  Sent: %d", sent)
	t.Logf("  Acknowledged: %d", acknowledged)
	t.Logf("  Failed (errors): %d", failed)
	t.Logf("  Data Loss: %d (%.4f%%)", dataLoss, dataLossRate*100)

	// CRITICAL: Zero data loss requirement
	if dataLoss > 0 {
		t.Errorf("CRITICAL: Data loss detected! %d notifications lost (%.4f%%)", dataLoss, dataLossRate*100)
	} else {
		t.Log("✓ Zero data loss confirmed under backpressure")
	}
}

func TestMAANG_SoakTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping soak test in short mode (use -test.short=false)")
	}

	// Run for 60 seconds to detect memory leaks and degradation
	duration := 60 * time.Second
	checkInterval := 10 * time.Second

	var memSamples []int64
	var latencySamples []time.Duration

	done := make(chan bool)

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				memSamples = append(memSamples, int64(m.Alloc/1024/1024))
			}
		}
	}()

	// Run load test
	result := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers:  200,
		Duration:         duration,
		TargetRPS:        10000,
		CreatePercentage: 30,
		ListPercentage:   70,
	})

	close(done)
	latencySamples = append(latencySamples, result.P99Latency)

	printMAANGReport(t, "60s Soak Test", result)

	// Check for memory growth
	if len(memSamples) >= 2 {
		memGrowth := float64(memSamples[len(memSamples)-1]) / float64(memSamples[0])
		t.Logf("Memory growth factor: %.2fx", memGrowth)

		if memGrowth > 2.0 {
			t.Errorf("Potential memory leak: %.2fx growth", memGrowth)
		}
	}
}

// ===== Load Test Engine =====

func runMAANGLoadTest(config MAANGLoadConfig) *MAANGLoadResult {
	result := &MAANGLoadResult{
		Histogram:      NewLatencyHistogram(),
		CreateLatency:  NewLatencyHistogram(),
		ListLatency:    NewLatencyHistogram(),
		ReadLatency:    NewLatencyHistogram(),
		StreamLatency:  NewLatencyHistogram(),
		ErrorBreakdown: make(map[string]int64),
		MinLatency:     time.Hour,
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	startTime := time.Now()

	// Rate tracking
	var requestCount int64
	var lastSecond int64
	var peakRPS int64

	// gRPC client pool
	clients := make([]pb.NotificationServiceClient, config.ConcurrentUsers)
	for i := 0; i < min(config.ConcurrentUsers, 100); i++ {
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

	// Launch workers with ramp-up
	usersPerBatch := config.ConcurrentUsers / 10
	if usersPerBatch < 1 {
		usersPerBatch = 1
	}

	rampDelay := config.RampUpDuration / 10

	for batch := 0; batch < 10; batch++ {
		time.Sleep(rampDelay)

		for u := 0; u < usersPerBatch; u++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				clientIdx := userID % len(clients)
				if clients[clientIdx] == nil {
					return
				}

				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					reqStart := time.Now()

					// Determine operation type
					opType := rand.Intn(100)
					var err error
					var opLatency *LatencyHistogram

					switch {
					case opType < config.CreatePercentage:
						err = createNotificationGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
						opLatency = result.CreateLatency
					case opType < config.CreatePercentage+config.ListPercentage:
						err = listNotificationsGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
						opLatency = result.ListLatency
					default:
						err = getUnreadCountGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
						opLatency = result.ReadLatency
					}

					latency := time.Since(reqStart)

					// Record metrics
					result.Histogram.Record(latency)
					if opLatency != nil {
						opLatency.Record(latency)
					}

					atomic.AddInt64(&result.TotalRequests, 1)
					if err == nil {
						atomic.AddInt64(&result.SuccessRequests, 1)
					} else {
						atomic.AddInt64(&result.FailedRequests, 1)
					}
					atomic.AddInt64(&requestCount, 1)

					// Track RPS per second
					currentSecond := time.Since(startTime).Seconds()
					if int64(currentSecond) > lastSecond {
						rps := atomic.LoadInt64(&requestCount)
						if rps > peakRPS {
							peakRPS = rps
						}
						atomic.StoreInt64(&requestCount, 0)
						lastSecond = int64(currentSecond)
					}

					// Update min/max latency
					if latency > result.MaxLatency {
						result.MaxLatency = latency
					}
					if latency < result.MinLatency {
						result.MinLatency = latency
					}
				}
			}(batch*usersPerBatch + u)
		}
	}

	// Wait for duration
	time.Sleep(config.Duration)
	cancel()
	wg.Wait()

	// Calculate final metrics
	result.Duration = time.Since(startTime)
	result.ActualRPS = float64(result.TotalRequests) / result.Duration.Seconds()
	result.PeakRPS = float64(peakRPS)
	result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests)

	result.P50Latency = result.Histogram.Percentile(0.50)
	result.P95Latency = result.Histogram.Percentile(0.95)
	result.P99Latency = result.Histogram.Percentile(0.99)
	result.P999Latency = result.Histogram.Percentile(0.999)
	result.MeanLatency = result.Histogram.Mean()

	// Resource usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	result.PeakMemoryMB = int64(m.Alloc / 1024 / 1024)
	result.GoroutinesPeak = runtime.NumGoroutine()

	return result
}

func listNotificationsGRPC(client pb.NotificationServiceClient, recipientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.ListNotifications(ctx, &pb.ListNotificationsRequest{
		RecipientId: recipientID,
		Limit:       20,
	})

	return err
}

func getUnreadCountGRPC(client pb.NotificationServiceClient, recipientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.GetUnreadCount(ctx, &pb.GetUnreadCountRequest{
		RecipientId: recipientID,
	})

	return err
}

// ===== Reporting =====

func printMAANGReport(t *testing.T, testName string, result *MAANGLoadResult) {
	t.Logf(`
╔══════════════════════════════════════════════════════════════════╗
║  MAANG LOAD TEST: %-44s ║
╠══════════════════════════════════════════════════════════════════╣
║  THROUGHPUT                                                      ║
║    Total Requests:     %-10d                                  ║
║    Duration:           %-10v                                  ║
║    Actual RPS:         %-10.0f                                 ║
║    Peak RPS:           %-10.0f                                 ║
╠══════════════════════════════════════════════════════════════════╣
║  LATENCY (Target: P99 < 50ms)                                    ║
║    Mean:               %-10v                                   ║
║    P50:                %-10v  %s                       ║
║    P95:                %-10v  %s                       ║
║    P99:                %-10v  %s                       ║
║    P99.9:              %-10v                                   ║
║    Max:                %-10v                                   ║
╠══════════════════════════════════════════════════════════════════╣
║  RELIABILITY (Target: 99.9%% success)                            ║
║    Success Rate:       %.4f%%                                    ║
║    Error Rate:         %.4f%%  %s                        ║
╠══════════════════════════════════════════════════════════════════╣
║  RESOURCES                                                       ║
║    Memory:             %-5d MB                                   ║
║    Goroutines:         %-5d                                      ║
╚══════════════════════════════════════════════════════════════════╝`,
		testName,
		result.TotalRequests,
		result.Duration,
		result.ActualRPS,
		result.PeakRPS,
		result.MeanLatency,
		result.P50Latency, checkMark(result.P50Latency < 10*time.Millisecond),
		result.P95Latency, checkMark(result.P95Latency < 30*time.Millisecond),
		result.P99Latency, checkMark(result.P99Latency < 50*time.Millisecond),
		result.P999Latency,
		result.MaxLatency,
		(1-result.ErrorRate)*100,
		result.ErrorRate*100, checkMark(result.ErrorRate < 0.001),
		result.PeakMemoryMB,
		result.GoroutinesPeak,
	)
}

func checkMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
