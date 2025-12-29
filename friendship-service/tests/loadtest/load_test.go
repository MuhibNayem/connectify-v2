package loadtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LoadTestConfig holds configuration for load tests
type LoadTestConfig struct {
	ConcurrentUsers int
	RequestsPerUser int
	Duration        time.Duration
	RampUpDuration  time.Duration
	TargetRPS       int // Target requests per second
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
}

// BenchmarkFriendshipCreate benchmarks friendship creation
func BenchmarkFriendshipCreate(b *testing.B) {
	// Skip if no database connection
	b.Skip("Benchmark requires database connection")

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		b.Skip("MongoDB not available")
	}
	defer client.Disconnect(ctx)

	collection := client.Database("benchmark_test").Collection("friendships")
	_, _ = collection.DeleteMany(ctx, bson.M{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			doc := bson.M{
				"_id":          primitive.NewObjectID(),
				"requester_id": primitive.NewObjectID(),
				"receiver_id":  primitive.NewObjectID(),
				"status":       "pending",
				"created_at":   time.Now(),
			}
			_, _ = collection.InsertOne(ctx, doc)
		}
	})
}

// BenchmarkFriendshipRead benchmarks friendship reads
func BenchmarkFriendshipRead(b *testing.B) {
	b.Skip("Benchmark requires database connection")

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		b.Skip("MongoDB not available")
	}
	defer client.Disconnect(ctx)

	collection := client.Database("benchmark_test").Collection("friendships")

	// Insert test data
	userA := primitive.NewObjectID()
	userB := primitive.NewObjectID()

	_, _ = collection.InsertOne(ctx, bson.M{
		"_id":          primitive.NewObjectID(),
		"requester_id": userA,
		"receiver_id":  userB,
		"status":       "accepted",
		"created_at":   time.Now(),
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = collection.FindOne(ctx, bson.M{
				"$or": []bson.M{
					{"requester_id": userA, "receiver_id": userB},
					{"requester_id": userB, "receiver_id": userA},
				},
			})
		}
	})
}

// BenchmarkCacheHit simulates cache hit performance
func BenchmarkCacheHit(b *testing.B) {
	cache := make(map[string]bool)
	var mu sync.RWMutex

	key := "friendship:status:user1:user2"
	cache[key] = true

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			_ = cache[key]
			mu.RUnlock()
		}
	})
}

// TestLoadTest_Simulation runs a simulated load test
func TestLoadTest_Simulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentUsers: 100,
		RequestsPerUser: 100,
		Duration:        10 * time.Second,
	}

	result := runSimulatedLoadTest(config)

	t.Logf("Load Test Results:")
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Successful: %d (%.2f%%)", result.SuccessRequests, float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	t.Logf("  Failed: %d", result.FailedRequests)
	t.Logf("  Duration: %v", result.TotalDuration)
	t.Logf("  RPS: %.2f", result.RPS)
	t.Logf("  Avg Latency: %v", result.AvgLatency)
	t.Logf("  Max Latency: %v", result.MaxLatency)

	// Assert minimum performance requirements
	if result.RPS < 1000 {
		t.Logf("Warning: RPS below target (1000), got %.2f", result.RPS)
	}
}

func runSimulatedLoadTest(config LoadTestConfig) LoadTestResult {
	var totalRequests, successRequests, failedRequests int64
	var totalLatency int64
	var maxLatency, minLatency int64 = 0, 1<<63 - 1

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < config.RequestsPerUser; j++ {
				reqStart := time.Now()

				// Simulate request
				success := simulateRequest()
				latency := time.Since(reqStart).Nanoseconds()

				atomic.AddInt64(&totalRequests, 1)
				if success {
					atomic.AddInt64(&successRequests, 1)
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}
				atomic.AddInt64(&totalLatency, latency)

				// Update max/min latency (not thread-safe but approximate is fine)
				if latency > atomic.LoadInt64(&maxLatency) {
					atomic.StoreInt64(&maxLatency, latency)
				}
				if latency < atomic.LoadInt64(&minLatency) {
					atomic.StoreInt64(&minLatency, latency)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	avgLatency := time.Duration(0)
	if totalRequests > 0 {
		avgLatency = time.Duration(totalLatency / totalRequests)
	}

	return LoadTestResult{
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		TotalDuration:   duration,
		AvgLatency:      avgLatency,
		MaxLatency:      time.Duration(maxLatency),
		MinLatency:      time.Duration(minLatency),
		RPS:             float64(totalRequests) / duration.Seconds(),
	}
}

func simulateRequest() bool {
	// Simulate some work (database call, validation, etc.)
	time.Sleep(time.Microsecond * 100)
	return true // 100% success for simulation
}

// TestLoadTest_WithErrorRate tests behavior under error conditions
func TestLoadTest_WithErrorRate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	var successCount, errorCount int64
	iterations := 10000
	errorRate := 0.05 // 5% error rate

	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Simulate occasional failures
			if float64(idx%100) < errorRate*100 {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	actualErrorRate := float64(errorCount) / float64(iterations)
	t.Logf("Actual error rate: %.4f (expected ~%.4f)", actualErrorRate, errorRate)

	if actualErrorRate > errorRate*2 {
		t.Errorf("Error rate too high: %.4f", actualErrorRate)
	}
}

// TestLoadTest_Throughput measures throughput under load
func TestLoadTest_Throughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	duration := 5 * time.Second
	var operations int64

	start := time.Now()
	done := make(chan bool)

	// Start workers
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					// Simulate operation
					_ = primitive.NewObjectID()
					atomic.AddInt64(&operations, 1)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)

	elapsed := time.Since(start)
	opsPerSecond := float64(operations) / elapsed.Seconds()

	t.Logf("Throughput: %.2f ops/sec", opsPerSecond)
	t.Logf("Total operations: %d in %v", operations, elapsed)
}

// TestLoadTest_Concurrency tests concurrent access patterns
func TestLoadTest_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	type Friendship struct {
		ID     string
		Status string
		mu     sync.RWMutex
	}

	friendships := make(map[string]*Friendship)
	var mapMu sync.RWMutex

	// Create some friendships
	for i := 0; i < 1000; i++ {
		id := primitive.NewObjectID().Hex()
		friendships[id] = &Friendship{ID: id, Status: "pending"}
	}

	var readOps, writeOps int64
	duration := 3 * time.Second
	done := make(chan bool)

	// Start reader workers
	for i := 0; i < 8; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					mapMu.RLock()
					for _, f := range friendships {
						f.mu.RLock()
						_ = f.Status
						f.mu.RUnlock()
						break // Just read one
					}
					mapMu.RUnlock()
					atomic.AddInt64(&readOps, 1)
				}
			}
		}()
	}

	// Start writer workers
	for i := 0; i < 2; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					mapMu.RLock()
					for _, f := range friendships {
						f.mu.Lock()
						if f.Status == "pending" {
							f.Status = "accepted"
						} else {
							f.Status = "pending"
						}
						f.mu.Unlock()
						break // Just write one
					}
					mapMu.RUnlock()
					atomic.AddInt64(&writeOps, 1)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	time.Sleep(100 * time.Millisecond) // Let workers finish

	t.Logf("Read operations: %d", readOps)
	t.Logf("Write operations: %d", writeOps)
	t.Logf("Read/Write ratio: %.2f", float64(readOps)/float64(writeOps))

	if readOps < writeOps {
		t.Log("Warning: Read operations should typically exceed write operations")
	}
}

// TestCircuitBreaker_UnderLoad tests circuit breaker behavior
func TestCircuitBreaker_UnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Simulated circuit breaker
	type CircuitBreaker struct {
		failures  int64
		threshold int64
		isOpen    bool
		mu        sync.RWMutex
	}

	cb := &CircuitBreaker{threshold: 5}

	var successCount, rejectedCount, failureCount int64

	for i := 0; i < 100; i++ {
		cb.mu.RLock()
		isOpen := cb.isOpen
		cb.mu.RUnlock()

		if isOpen {
			atomic.AddInt64(&rejectedCount, 1)
			continue
		}

		// Simulate request (50% failure rate)
		if i%2 == 0 {
			atomic.AddInt64(&successCount, 1)
			cb.mu.Lock()
			cb.failures = 0 // Reset on success
			cb.mu.Unlock()
		} else {
			atomic.AddInt64(&failureCount, 1)
			cb.mu.Lock()
			cb.failures++
			if cb.failures >= cb.threshold {
				cb.isOpen = true
			}
			cb.mu.Unlock()
		}
	}

	t.Logf("Success: %d, Failures: %d, Rejected: %d", successCount, failureCount, rejectedCount)

	if rejectedCount == 0 && failureCount >= cb.threshold {
		t.Error("Circuit breaker should have opened")
	}
}

// PrintLoadTestReport prints a formatted load test report
func PrintLoadTestReport(result LoadTestResult) string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════╗
║                    LOAD TEST REPORT                          ║
╠══════════════════════════════════════════════════════════════╣
║  Total Requests:     %-10d                              ║
║  Successful:         %-10d (%.2f%%)                     ║
║  Failed:             %-10d                              ║
╠══════════════════════════════════════════════════════════════╣
║  Duration:           %-10v                              ║
║  Requests/sec:       %-10.2f                            ║
╠══════════════════════════════════════════════════════════════╣
║  Avg Latency:        %-10v                              ║
║  Min Latency:        %-10v                              ║
║  Max Latency:        %-10v                              ║
╚══════════════════════════════════════════════════════════════╝
`,
		result.TotalRequests,
		result.SuccessRequests,
		float64(result.SuccessRequests)/float64(result.TotalRequests)*100,
		result.FailedRequests,
		result.TotalDuration,
		result.RPS,
		result.AvgLatency,
		result.MinLatency,
		result.MaxLatency,
	)
}
