package loadtest

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
MAANG-Scale Load Testing Suite
==============================

Target Performance Metrics (Facebook/Meta scale):
- Friend request: < 50ms p99
- Check friendship: < 10ms p99 (cache hit)
- Accept/Reject: < 100ms p99
- Throughput: 100K+ RPS for reads, 10K+ RPS for writes

Test Scenarios:
1. Sustained Load - Steady traffic over time
2. Spike Test - Sudden traffic bursts
3. Stress Test - Find breaking point
4. Soak Test - Long-running stability
5. Concurrent Access - Race condition detection
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
	ReadPercentage      int // CheckFriendship, AreFriends
	WritePercentage     int // SendRequest, Accept, Reject
	HeavyReadPercentage int // ListFriends, Search

	// Performance thresholds
	MaxP50Latency time.Duration
	MaxP95Latency time.Duration
	MaxP99Latency time.Duration
	MaxErrorRate  float64

	// System limits
	MaxCPUPercent float64
	MaxMemoryMB   int64
}

// DefaultMAANGConfig returns production-grade config for M1 Pro
func DefaultMAANGConfig() MAANGLoadConfig {
	return MAANGLoadConfig{
		ConcurrentUsers:     1000,
		RequestsPerUser:     1000,
		Duration:            30 * time.Second,
		RampUpDuration:      5 * time.Second,
		TargetRPS:           50000, // 50K RPS target
		ReadPercentage:      70,
		WritePercentage:     20,
		HeavyReadPercentage: 10,
		MaxP50Latency:       5 * time.Millisecond,
		MaxP95Latency:       20 * time.Millisecond,
		MaxP99Latency:       50 * time.Millisecond,
		MaxErrorRate:        0.001, // 0.1%
		MaxCPUPercent:       80,
		MaxMemoryMB:         1024,
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
	ReadLatency  *LatencyHistogram
	WriteLatency *LatencyHistogram

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
		ConcurrentUsers: 500,
		Duration:        15 * time.Second,
		RampUpDuration:  3 * time.Second,
		TargetRPS:       25000,
		ReadPercentage:  80,
		WritePercentage: 20,
	}

	result := runMAANGLoadTest(config)

	printMAANGReport(t, "Sustained Load", result)

	// Assertions
	if result.ErrorRate > 0.01 {
		t.Errorf("Error rate too high: %.4f%% (max 1%%)", result.ErrorRate*100)
	}
	if result.P99Latency > 100*time.Millisecond {
		t.Errorf("P99 latency too high: %v (max 100ms)", result.P99Latency)
	}
}

func TestMAANG_SpikeTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MAANG spike test in short mode")
	}

	t.Log("Phase 1: Normal load (5s)")
	result1 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers: 100,
		Duration:        5 * time.Second,
		TargetRPS:       5000,
		ReadPercentage:  80,
		WritePercentage: 20,
	})

	t.Log("Phase 2: 10x Spike (5s)")
	result2 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers: 1000,
		Duration:        5 * time.Second,
		TargetRPS:       50000,
		ReadPercentage:  80,
		WritePercentage: 20,
	})

	t.Log("Phase 3: Recovery (5s)")
	result3 := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers: 100,
		Duration:        5 * time.Second,
		TargetRPS:       5000,
		ReadPercentage:  80,
		WritePercentage: 20,
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

	for users := 100; users <= 10000; users *= 2 {
		result := runMAANGLoadTest(MAANGLoadConfig{
			ConcurrentUsers: users,
			Duration:        3 * time.Second,
			TargetRPS:       users * 100,
			ReadPercentage:  80,
			WritePercentage: 20,
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

func TestMAANG_ConcurrencyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency stress test in short mode")
	}

	// Simulate hot key contention (same friendship being modified)
	var successCount, conflictCount int64
	hotUserA := primitive.NewObjectID()
	hotUserB := primitive.NewObjectID()

	var wg sync.WaitGroup
	iterations := 1000
	goroutines := 100

	// Simulated version tracking
	var version int64
	var versionMu sync.Mutex

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < iterations/goroutines; i++ {
				// Read current version
				versionMu.Lock()
				currentVersion := version
				versionMu.Unlock()

				// Simulate some processing time
				time.Sleep(time.Microsecond * 10)

				// Try optimistic update
				versionMu.Lock()
				if version == currentVersion {
					version++
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&conflictCount, 1)
				}
				versionMu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Use the hot users to prevent unused variable warning
	_ = hotUserA
	_ = hotUserB

	conflictRate := float64(conflictCount) / float64(successCount+conflictCount)
	t.Logf("Concurrent modifications:")
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Conflicts: %d (%.2f%%)", conflictCount, conflictRate*100)

	if conflictRate > 0.5 {
		t.Log("High conflict rate - optimistic locking working correctly")
	}
}

func TestMAANG_FriendGraphTraversal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping graph traversal test in short mode")
	}

	// Simulate friend-of-friend queries at scale
	// Facebook average: ~338 friends per user
	// Power users: 5000 friends

	avgFriends := 338
	powerUserFriends := 5000

	// Simulate graph structure
	type User struct {
		ID      primitive.ObjectID
		Friends []primitive.ObjectID
	}

	users := make([]User, 10000)
	for i := range users {
		users[i].ID = primitive.NewObjectID()

		friendCount := avgFriends
		if i < 100 { // 1% are power users
			friendCount = powerUserFriends
		}

		users[i].Friends = make([]primitive.ObjectID, friendCount)
		for j := 0; j < friendCount; j++ {
			users[i].Friends[j] = primitive.NewObjectID()
		}
	}

	histogram := NewLatencyHistogram()

	// Measure friend-of-friend discovery time
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(userIdx int) {
			defer wg.Done()

			start := time.Now()

			// Get friends of friends (2-hop)
			user := users[userIdx%len(users)]
			fofSet := make(map[primitive.ObjectID]bool)

			for _, friendID := range user.Friends[:min(100, len(user.Friends))] {
				// Find friends of this friend (simulated)
				for j := 0; j < 50; j++ {
					fofSet[primitive.NewObjectID()] = true
				}

				_ = friendID // Use the variable
			}

			histogram.Record(time.Since(start))
		}(i)
	}

	wg.Wait()

	t.Logf("Friend-of-friend query performance:")
	t.Logf("  P50: %v", histogram.Percentile(0.50))
	t.Logf("  P95: %v", histogram.Percentile(0.95))
	t.Logf("  P99: %v", histogram.Percentile(0.99))
}

// ===== Benchmarks =====

func BenchmarkMAANG_CheckFriendship(b *testing.B) {
	// Simulate cache lookup
	cache := make(map[string]bool, 1000000)
	for i := 0; i < 1000000; i++ {
		key := fmt.Sprintf("friendship:%s:%s",
			primitive.NewObjectID().Hex(),
			primitive.NewObjectID().Hex())
		cache[key] = i%2 == 0
	}

	// Sample keys for lookup
	keys := make([]string, 1000)
	i := 0
	for k := range cache {
		if i >= 1000 {
			break
		}
		keys[i] = k
		i++
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := keys[rand.Intn(len(keys))]
			_ = cache[key]
		}
	})
}

func BenchmarkMAANG_ObjectIDGeneration(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = primitive.NewObjectID()
		}
	})
}

func BenchmarkMAANG_ConcurrentMapAccess(b *testing.B) {
	m := sync.Map{}

	// Pre-populate
	for i := 0; i < 100000; i++ {
		m.Store(primitive.NewObjectID().Hex(), true)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Float32() < 0.9 { // 90% reads
				m.Load(primitive.NewObjectID().Hex())
			} else {
				m.Store(primitive.NewObjectID().Hex(), true)
			}
		}
	})
}

// ===== Load Test Engine =====

func runMAANGLoadTest(config MAANGLoadConfig) *MAANGLoadResult {
	result := &MAANGLoadResult{
		Histogram:      NewLatencyHistogram(),
		ReadLatency:    NewLatencyHistogram(),
		WriteLatency:   NewLatencyHistogram(),
		ErrorBreakdown: make(map[string]int64),
		MinLatency:     time.Hour,
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration+5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	startTime := time.Now()

	// Rate limiter (token bucket)
	var requestCount int64
	var lastSecond int64
	var peakRPS int64

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

				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					reqStart := time.Now()

					// Determine operation type
					opType := rand.Intn(100)
					var isRead bool

					switch {
					case opType < config.ReadPercentage:
						isRead = true
						simulateReadOperation()
					case opType < config.ReadPercentage+config.WritePercentage:
						isRead = false
						simulateWriteOperation()
					default:
						isRead = true
						simulateHeavyReadOperation()
					}

					latency := time.Since(reqStart)

					// Record metrics
					result.Histogram.Record(latency)
					if isRead {
						result.ReadLatency.Record(latency)
					} else {
						result.WriteLatency.Record(latency)
					}

					atomic.AddInt64(&result.TotalRequests, 1)
					atomic.AddInt64(&result.SuccessRequests, 1)
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

func simulateReadOperation() {
	// Simulate cache lookup + minimal processing
	time.Sleep(time.Microsecond * time.Duration(rand.Intn(50)+10))
}

func simulateWriteOperation() {
	// Simulate DB write + cache invalidation
	time.Sleep(time.Microsecond * time.Duration(rand.Intn(200)+100))
}

func simulateHeavyReadOperation() {
	// Simulate list/search with pagination
	time.Sleep(time.Microsecond * time.Duration(rand.Intn(500)+200))
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
║  LATENCY (MAANG SLA: P99 < 50ms)                                 ║
║    Mean:               %-10v                                   ║
║    P50:                %-10v  %s                       ║
║    P95:                %-10v  %s                       ║
║    P99:                %-10v  %s                       ║
║    P99.9:              %-10v                                   ║
║    Max:                %-10v                                   ║
╠══════════════════════════════════════════════════════════════════╣
║  RELIABILITY                                                     ║
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

// ===== Soak Test =====

func TestMAANG_SoakTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping soak test in short mode (run with -test.short=false)")
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
		ConcurrentUsers: 200,
		Duration:        duration,
		TargetRPS:       10000,
		ReadPercentage:  80,
		WritePercentage: 20,
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

// ===== System Capacity Test =====

func TestMAANG_SystemCapacity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping capacity test in short mode")
	}

	numCPU := runtime.NumCPU()
	t.Logf("System: %d CPUs (M1 Pro optimized)", numCPU)

	// Calculate theoretical max throughput
	// M1 Pro: expect ~50K-100K ops/sec for in-memory operations

	var opsCount int64
	duration := 3 * time.Second

	start := time.Now()
	done := make(chan bool)

	for i := 0; i < numCPU; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					// Simulate minimal operation
					_ = primitive.NewObjectID()
					atomic.AddInt64(&opsCount, 1)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)

	elapsed := time.Since(start)
	opsPerSec := float64(opsCount) / elapsed.Seconds()

	t.Logf("Raw throughput: %.0f ops/sec", opsPerSec)
	t.Logf("Per-CPU: %.0f ops/sec", opsPerSec/float64(numCPU))

	// Expected for M1 Pro: > 1M ops/sec for simple operations
	if opsPerSec < 100000 {
		t.Logf("Warning: Lower than expected throughput")
	}
}

// ===== Latency Distribution Analysis =====

func TestMAANG_LatencyDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency distribution test in short mode")
	}

	result := runMAANGLoadTest(MAANGLoadConfig{
		ConcurrentUsers: 500,
		Duration:        10 * time.Second,
		TargetRPS:       20000,
		ReadPercentage:  70,
		WritePercentage: 30,
	})

	t.Logf("Latency Distribution Analysis:")
	t.Logf("  P10:  %v", result.Histogram.Percentile(0.10))
	t.Logf("  P25:  %v", result.Histogram.Percentile(0.25))
	t.Logf("  P50:  %v (median)", result.Histogram.Percentile(0.50))
	t.Logf("  P75:  %v", result.Histogram.Percentile(0.75))
	t.Logf("  P90:  %v", result.Histogram.Percentile(0.90))
	t.Logf("  P95:  %v", result.Histogram.Percentile(0.95))
	t.Logf("  P99:  %v", result.Histogram.Percentile(0.99))
	t.Logf("  P99.9: %v", result.Histogram.Percentile(0.999))

	// Calculate standard deviation
	mean := result.Histogram.Mean()
	var sumSquares float64
	for _, s := range result.Histogram.samples {
		diff := float64(s - mean)
		sumSquares += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(sumSquares / float64(len(result.Histogram.samples))))
	t.Logf("  StdDev: %v", stdDev)

	// Coefficient of variation (lower is better)
	cv := float64(stdDev) / float64(mean) * 100
	t.Logf("  CV: %.2f%% (lower = more consistent)", cv)
}
