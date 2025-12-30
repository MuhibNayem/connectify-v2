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

/*
Capacity Planning Test - Gradual Load Increase Until Failure

This test incrementally increases load to find the exact breaking point
and generates detailed performance analysis at each step.

Purpose:
- Identify maximum sustainable throughput
- Detect performance degradation patterns
- Provide capacity planning data
- Validate horizontal scalability limits
*/

// CapacityLevel represents metrics at a specific load level
type CapacityLevel struct {
	ConcurrentUsers int
	Duration        time.Duration
	TotalRequests   int64
	SuccessRate     float64
	ErrorRate       float64
	ActualRPS       float64
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	MeanLatency     time.Duration
	MaxLatency      time.Duration
	MemoryMB        int64
	Goroutines      int
	Degraded        bool
	DegradationMsg  string
}

// CapacityTestResult holds the complete capacity analysis
type CapacityTestResult struct {
	Levels              []CapacityLevel
	MaxStableUsers      int
	MaxStableRPS        float64
	BreakingPoint       int
	PeakRPS             float64
	OptimalUsers        int // Best RPS/latency ratio
	RecommendedCapacity int // 70% of max for safety margin
}

// TestCapacity_GradualIncrease runs incremental load test until degradation
func TestCapacity_GradualIncrease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping capacity test in short mode")
	}

	config := CapacityTestConfig{
		StartUsers:     10,
		MaxUsers:       5000,
		StepMultiplier: 1.5, // 50% increase each step
		StepDuration:   15 * time.Second,
		StabilizePause: 2 * time.Second,

		// Degradation thresholds
		MaxErrorRate:  0.01, // 1%
		MaxP99Latency: 200 * time.Millisecond,
		MinRPSGrowth:  0.3, // Must achieve 30% of expected growth

		// Workload
		CreatePercentage: 30,
		ListPercentage:   70,
	}

	result := runCapacityTest(t, config)

	printCapacityReport(t, result)

	// Assertions
	if result.MaxStableUsers < 100 {
		t.Errorf("System capacity too low: only %d stable users", result.MaxStableUsers)
	}

	if result.MaxStableRPS < 5000 {
		t.Logf("Warning: Peak RPS below target (5K), got %.0f", result.MaxStableRPS)
	}
}

// TestCapacity_DetailedAnalysis runs slower, more detailed capacity test
func TestCapacity_DetailedAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping detailed capacity test in short mode")
	}

	config := CapacityTestConfig{
		StartUsers:       5,
		MaxUsers:         2000,
		StepMultiplier:   1.25, // Smaller steps for precision
		StepDuration:     20 * time.Second,
		StabilizePause:   3 * time.Second,
		MaxErrorRate:     0.005, // 0.5% (stricter)
		MaxP99Latency:    100 * time.Millisecond,
		MinRPSGrowth:     0.5,
		CreatePercentage: 20,
		ListPercentage:   80,
	}

	result := runCapacityTest(t, config)
	printCapacityReport(t, result)
	printCapacityGraph(t, result)
}

// CapacityTestConfig holds capacity test configuration
type CapacityTestConfig struct {
	StartUsers       int
	MaxUsers         int
	StepMultiplier   float64
	StepDuration     time.Duration
	StabilizePause   time.Duration
	MaxErrorRate     float64
	MaxP99Latency    time.Duration
	MinRPSGrowth     float64
	CreatePercentage int
	ListPercentage   int
}

func runCapacityTest(t *testing.T, config CapacityTestConfig) *CapacityTestResult {
	result := &CapacityTestResult{
		Levels: make([]CapacityLevel, 0),
	}

	currentUsers := config.StartUsers
	var previousRPS float64
	consecutiveFailures := 0

	t.Logf("\n🔬 Starting Capacity Test: %d → %d users", config.StartUsers, config.MaxUsers)
	t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for currentUsers <= config.MaxUsers {
		t.Logf("\n📊 Load Level: %d concurrent users", currentUsers)

		// Run load test at this level
		level := runCapacityLevel(config, currentUsers)

		// Check for degradation
		level.Degraded, level.DegradationMsg = detectDegradation(level, config, previousRPS)

		result.Levels = append(result.Levels, level)

		// Print immediate feedback
		status := "✓ HEALTHY"
		if level.Degraded {
			status = "⚠ DEGRADED"
			consecutiveFailures++
		} else {
			consecutiveFailures = 0
			result.MaxStableUsers = currentUsers
			result.MaxStableRPS = level.ActualRPS
		}

		t.Logf("   RPS: %.0f | P99: %v | Errors: %.3f%% | %s",
			level.ActualRPS, level.P99Latency, level.ErrorRate*100, status)

		if level.Degraded {
			t.Logf("   ⮕ Degradation: %s", level.DegradationMsg)
		}

		// Stop if 2 consecutive failures
		if consecutiveFailures >= 2 {
			result.BreakingPoint = currentUsers
			t.Logf("\n🛑 Breaking point detected at %d users", currentUsers)
			break
		}

		// Track peak RPS
		if level.ActualRPS > result.PeakRPS {
			result.PeakRPS = level.ActualRPS
		}

		previousRPS = level.ActualRPS

		// Pause before next level
		if currentUsers < config.MaxUsers {
			time.Sleep(config.StabilizePause)
		}

		// Calculate next step
		nextUsers := int(float64(currentUsers) * config.StepMultiplier)
		if nextUsers == currentUsers {
			nextUsers++ // Ensure we always increment
		}
		currentUsers = nextUsers
	}

	// Calculate optimal and recommended capacity
	result.OptimalUsers = findOptimalCapacity(result.Levels)
	result.RecommendedCapacity = int(float64(result.MaxStableUsers) * 0.7)

	return result
}

func runCapacityLevel(config CapacityTestConfig, users int) CapacityLevel {
	ctx, cancel := context.WithTimeout(context.Background(), config.StepDuration+5*time.Second)
	defer cancel()

	histogram := NewLatencyHistogram()
	var totalRequests, successRequests, failedRequests int64
	var maxLatency time.Duration

	// Create gRPC clients
	clients := make([]pb.NotificationServiceClient, min(users, 100))
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

	// Launch workers
	for i := 0; i < users; i++ {
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

				// Determine operation
				var err error
				var errMsg string
				opType := int(atomic.AddInt64(&totalRequests, 1)) % 100
				if opType < config.CreatePercentage {
					err = createNotificationGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
					if err != nil {
						errMsg = fmt.Sprintf("Create failed: %v", err)
					}
				} else {
					err = listNotificationsGRPC(clients[clientIdx], fmt.Sprintf("user-%d", userID))
					if err != nil {
						errMsg = fmt.Sprintf("List failed: %v", err)
					}
				}

				latency := time.Since(reqStart)
				histogram.Record(latency)

				if err == nil {
					atomic.AddInt64(&successRequests, 1)
				} else {
					atomic.AddInt64(&failedRequests, 1)
					// Log first few errors for debugging
					if failedRequests < 3 {
						fmt.Printf("   DEBUG: %s\n", errMsg)
					}
				}

				// Track max latency
				if latency > maxLatency {
					maxLatency = latency
				}
			}
		}(i)
	}

	// Wait for test duration
	time.Sleep(config.StepDuration)
	cancel()
	wg.Wait()

	duration := time.Since(startTime)

	// Calculate metrics
	total := totalRequests
	success := successRequests
	errorRate := float64(failedRequests) / float64(total)
	successRate := float64(success) / float64(total)

	return CapacityLevel{
		ConcurrentUsers: users,
		Duration:        duration,
		TotalRequests:   total,
		SuccessRate:     successRate,
		ErrorRate:       errorRate,
		ActualRPS:       float64(total) / duration.Seconds(),
		P50Latency:      histogram.Percentile(0.50),
		P95Latency:      histogram.Percentile(0.95),
		P99Latency:      histogram.Percentile(0.99),
		MeanLatency:     histogram.Mean(),
		MaxLatency:      maxLatency,
	}
}

func detectDegradation(level CapacityLevel, config CapacityTestConfig, previousRPS float64) (bool, string) {
	// Error rate too high
	if level.ErrorRate > config.MaxErrorRate {
		return true, fmt.Sprintf("Error rate %.3f%% exceeds threshold %.3f%%",
			level.ErrorRate*100, config.MaxErrorRate*100)
	}

	// Latency too high
	if level.P99Latency > config.MaxP99Latency {
		return true, fmt.Sprintf("P99 latency %v exceeds threshold %v",
			level.P99Latency, config.MaxP99Latency)
	}

	// RPS growth stalled (plateau detection)
	if previousRPS > 0 {
		expectedGrowth := previousRPS * (config.StepMultiplier - 1.0)
		actualGrowth := level.ActualRPS - previousRPS
		growthRatio := actualGrowth / expectedGrowth

		if growthRatio < config.MinRPSGrowth {
			return true, fmt.Sprintf("RPS growth stalled (%.1f%% of expected)",
				growthRatio*100)
		}
	}

	return false, ""
}

func findOptimalCapacity(levels []CapacityLevel) int {
	if len(levels) == 0 {
		return 0
	}

	bestScore := 0.0
	optimalUsers := 0

	for _, level := range levels {
		if level.Degraded {
			continue
		}

		// Score = RPS / (P99_latency_ms + error_rate%)
		latencyPenalty := float64(level.P99Latency.Milliseconds())
		errorPenalty := level.ErrorRate * 1000 // Scale error rate
		score := level.ActualRPS / (latencyPenalty + errorPenalty + 1)

		if score > bestScore {
			bestScore = score
			optimalUsers = level.ConcurrentUsers
		}
	}

	return optimalUsers
}

// ===== Reporting =====

func printCapacityReport(t *testing.T, result *CapacityTestResult) {
	t.Logf(`
╔══════════════════════════════════════════════════════════════════╗
║                    CAPACITY TEST REPORT                          ║
╠══════════════════════════════════════════════════════════════════╣
║  CAPACITY LIMITS                                                 ║
║    Max Stable Users:      %-6d                                  ║
║    Breaking Point:        %-6d users                            ║
║    Optimal Users:         %-6d (best RPS/latency ratio)         ║
║    Recommended Capacity:  %-6d (70%% safety margin)             ║
╠══════════════════════════════════════════════════════════════════╣
║  THROUGHPUT                                                      ║
║    Max Stable RPS:        %-10.0f                               ║
║    Peak RPS:              %-10.0f                               ║
╠══════════════════════════════════════════════════════════════════╣
║  LOAD LEVELS TESTED                                              ║`,
		result.MaxStableUsers,
		result.BreakingPoint,
		result.OptimalUsers,
		result.RecommendedCapacity,
		result.MaxStableRPS,
		result.PeakRPS,
	)

	for _, level := range result.Levels {
		status := "✓"
		if level.Degraded {
			status = "✗"
		}

		t.Logf("║    %s Users: %-4d | RPS: %-6.0f | P99: %-6v | Err: %.2f%%",
			status,
			level.ConcurrentUsers,
			level.ActualRPS,
			level.P99Latency,
			level.ErrorRate*100,
		)
	}

	t.Logf("╚══════════════════════════════════════════════════════════════════╝")

	// Recommendations
	t.Logf("\n📋 RECOMMENDATIONS:")
	if result.MaxStableUsers > 0 && result.OptimalUsers > 0 {
		podsNeeded := result.RecommendedCapacity/result.OptimalUsers + 1
		perPodRPS := result.MaxStableRPS / float64(result.MaxStableUsers) * float64(result.OptimalUsers)
		t.Logf("   • Deploy with %d pods for safety margin", podsNeeded)
		t.Logf("   • Set horizontal pod autoscaler (HPA) threshold at 70%% utilization")
		t.Logf("   • Expected per-pod capacity: %.0f RPS", perPodRPS)
	} else {
		t.Logf("   ⚠ WARNING: No stable capacity found!")
		t.Logf("   • All load levels failed connection/performance tests")
		t.Logf("   • Check server logs for errors")
		t.Logf("   • Verify gRPC server is running on %s", grpcAddr)
		t.Logf("   • Ensure database and queue broker are accessible")
	}
}

func printCapacityGraph(t *testing.T, result *CapacityTestResult) {
	t.Logf("\n📈 PERFORMANCE CURVE (RPS vs Load):")
	t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	maxRPS := result.PeakRPS
	for _, level := range result.Levels {
		bars := int((level.ActualRPS / maxRPS) * 50)
		graph := ""
		for i := 0; i < bars; i++ {
			if level.Degraded {
				graph += "▓"
			} else {
				graph += "█"
			}
		}

		marker := " "
		if level.ConcurrentUsers == result.OptimalUsers {
			marker = "⭐"
		} else if level.ConcurrentUsers == result.MaxStableUsers {
			marker = "✓"
		} else if level.Degraded {
			marker = "✗"
		}

		t.Logf("%s %4d users │ %-50s │ %.0f RPS",
			marker,
			level.ConcurrentUsers,
			graph,
			level.ActualRPS,
		)
	}

	t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	t.Logf("Legend: ⭐ Optimal | ✓ Max Stable | ✗ Degraded | █ Healthy | ▓ Degraded")
}
