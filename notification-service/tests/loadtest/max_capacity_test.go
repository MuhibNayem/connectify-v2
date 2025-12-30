package loadtest

import (
	"testing"
	"time"
)

// TestMaxCapacity_FindBreakingPoint runs incremental tests to find max capacity
func TestMaxCapacity_FindBreakingPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping max capacity test in short mode")
	}

	userLevels := []int{500, 1000, 2000, 3000, 5000}
	results := make([]RealisticLoadResult, 0)

	t.Logf("\n🔬 Finding Single Instance Maximum Capacity")
	t.Logf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for _, users := range userLevels {
		t.Logf("📊 Testing %d concurrent users...", users)

		config := RealisticLoadConfig{
			ConcurrentUsers:  users,
			RequestsPerUser:  50, // Fewer requests for faster test
			RequestPacing:    100 * time.Millisecond,
			Duration:         30 * time.Second,
			CreatePercentage: 30,
			ListPercentage:   70,
		}

		result := runRealisticLoadTest(config)
		results = append(results, result)

		status := "✓ HEALTHY"
		if result.ErrorRate > 0.01 {
			status = "✗ DEGRADED (errors)"
		} else if result.P99Latency > 200*time.Millisecond {
			status = "⚠ DEGRADED (latency)"
		}

		t.Logf("   RPS: %.0f | P99: %v | Errors: %.2f%% | %s\n",
			result.ActualRPS, result.P99Latency, result.ErrorRate*100, status)

		// Short pause between tests
		time.Sleep(2 * time.Second)
	}

	// Print summary
	t.Logf("\n╔══════════════════════════════════════════════════════════════════╗")
	t.Logf("║            SINGLE INSTANCE CAPACITY ANALYSIS                     ║")
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  Users │   RPS  │  P50   │  P99   │ Errors │ Status            ║")
	t.Logf("╠══════════════════════════════════════════════════════════════════╣")

	maxHealthyUsers := 0
	maxHealthyRPS := 0.0

	for i, result := range results {
		users := userLevels[i]
		status := "✓ Healthy"

		if result.ErrorRate > 0.01 {
			status = "✗ High Errors"
		} else if result.P99Latency > 200*time.Millisecond {
			status = "⚠ High Latency"
		} else {
			maxHealthyUsers = users
			maxHealthyRPS = result.ActualRPS
		}

		t.Logf("║  %-5d │ %6.0f │ %6v │ %6v │ %5.2f%% │ %-18s║",
			users,
			result.ActualRPS,
			result.P50Latency,
			result.P99Latency,
			result.ErrorRate*100,
			status,
		)
	}

	t.Logf("╠══════════════════════════════════════════════════════════════════╣")
	t.Logf("║  MAXIMUM CAPACITY                                                ║")
	t.Logf("║    Max Concurrent Users:  %-6d                                 ║", maxHealthyUsers)
	t.Logf("║    Max Sustained RPS:     %-10.0f                            ║", maxHealthyRPS)
	t.Logf("║    Recommended (70%%):     %-10.0f RPS                        ║", maxHealthyRPS*0.7)
	t.Logf("╚══════════════════════════════════════════════════════════════════╝")

	if maxHealthyUsers == 0 {
		t.Errorf("Failed to find healthy capacity level")
	}
}
