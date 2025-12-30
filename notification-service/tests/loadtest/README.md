# Load and Stress Testing Suite

Comprehensive load testing for the notification service following MAANF (Meta/Amazon/Apple/Netflix/Google) scale standards.

## Test Files

### 1. `load_test.go` - Basic Load Tests
Standard load tests for HTTP and gRPC endpoints:
- `TestLoadTest_HTTP_CreateNotification` - HTTP create endpoint
- `TestLoadTest_gRPC_CreateNotification` - gRPC unary create
- `TestLoadTest_gRPC_StreamNotifications` - gRPC streaming (50K+ RPS target)
- `TestLoadTest_MixedWorkload` - Realistic 70/20/10 read/write/heavy mix

### 2. `maanf_load_test.go` - MAANF-Scale Stress Tests
Production-grade stress testing:
- `TestMAANG_SustainedLoad` - 15s sustained load test
- `TestMAANG_SpikeTest` - 10x traffic spike simulation
- `TestMAANG_StressTest` - Find breaking point (auto-scaling)
- `TestMAANG_StreamingBackpressureTest` - **CRITICAL: Zero data loss validation**
- `TestMAANG_SoakTest` - 60s memory leak detection

### 3. `capacity_test.go` - Incremental Capacity Planning
Gradual load increase with degradation detection:
- `TestCapacity_GradualIncrease` - Find breaking point automatically
- `TestCapacity_DetailedAnalysis` - Precision capacity analysis with performance graphs

**Key Features:**
- Automatically detects system degradation (error rate, P99 latency, RPS plateau)
- Identifies optimal capacity (best RPS/latency ratio)
- Provides deployment recommendations with safety margins  
- Generates performance curve visualization

## Prerequisites

### 1. Start the Notification Service
```bash
# Terminal 1 - Start server
cd /Users/a.k.mmuhibullahnayem/Developer/connectify-v2/notification-service
go run cmd/server/main.go
```

### 2. Verify RabbitMQ/Kafka is Running
```bash
# Check RabbitMQ
curl -u guest:guest http://localhost:15672/api/overview

# Or check Kafka
kafka-topics.sh --bootstrap-server localhost:9092 --list
```

## Running Tests

### Quick Smoke Test (skip long tests)
```bash
go test -v ./tests/loadtest -short
```

### Basic Load Tests
```bash
# HTTP endpoints
go test -v ./tests/loadtest -run TestLoadTest_HTTP

# gRPC endpoints
go test -v ./tests/loadtest -run TestLoadTest_gRPC

# Mixed workload
go test -v ./tests/loadtest -run TestLoadTest_MixedWorkload
```

### MAANF-Scale Tests
```bash
# Sustained load (15s)
go test -v ./tests/loadtest -run TestMAANG_SustainedLoad

# Spike test (3 phases)
go test -v ./tests/loadtest -run TestMAANG_SpikeTest

# Stress test (find breaking point)
go test -v ./tests/loadtest -run TestMAANG_StressTest -timeout 5m

# CRITICAL: Backpressure test (zero data loss validation)
go test -v ./tests/loadtest -run TestMAANG_StreamingBackpressureTest

# Soak test (60s memory leak detection)
go test -v ./tests/loadtest -run TestMAANG_SoakTest -timeout 5m
```

### Capacity Planning Tests (Gradual Load Increase)
```bash
# Automatic capacity test (finds breaking point)
go test -v ./tests/loadtest -run TestCapacity_GradualIncrease -timeout 10m

# Detailed analysis with graphs (slower, more precise)
go test -v ./tests/loadtest -run TestCapacity_DetailedAnalysis -timeout 15m
```

**Expected Output:**
```
🔬 Starting Capacity Test: 10 → 5000 users
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 Load Level: 10 concurrent users
   RPS: 2500 | P99: 8ms | Errors: 0.000% | ✓ HEALTHY

📊 Load Level: 500 concurrent users
   RPS: 10200 | P99: 45ms | Errors: 0.080% | ✓ HEALTHY

📊 Load Level: 750 concurrent users
   RPS: 12500 | P99: 185ms | Errors: 1.200% | ⚠ DEGRADED
   ⮕ Degradation: Error rate 1.200% exceeds threshold 1.000%

🛑 Breaking point detected at 750 users

╔══════════════════════════════════════════════════════════════════╗
║                    CAPACITY TEST REPORT                          ║
╠══════════════════════════════════════════════════════════════════╣
║  CAPACITY LIMITS                                                 ║
║    Max Stable Users:      500                                    ║
║    Breaking Point:        750   users                            ║
║    Optimal Users:         350   (best RPS/latency ratio)         ║
║    Recommended Capacity:  350   (70% safety margin)              ║
╠══════════════════════════════════════════════════════════════════╣
║  THROUGHPUT                                                      ║
║    Max Stable RPS:        10500                                  ║
║    Peak RPS:              12500                                  ║
╚══════════════════════════════════════════════════════════════════╝

📋 RECOMMENDATIONS:
   • Deploy with 2 pods for safety margin
   • Set horizontal pod autoscaler (HPA) threshold at 70% utilization
   • Expected per-pod capacity: 7350 RPS
```

### Run All Tests (WARNING: Takes 5-10 minutes)
```bash
go test -v ./tests/loadtest -timeout 15m
```

## Performance Targets

### Latency SLAs
- **P50**: < 10ms
- **P95**: < 30ms
- **P99**: < 50ms (MAANF standard)
- **P99.9**: < 100ms

### Throughput
- **HTTP Create**: 1K+ RPS
- **gRPC Unary**: 2K+ RPS
- **gRPC Streaming**: **50K+ RPS** (micro-batching)
- **Mixed Workload**: 5K+ RPS

### Reliability
- **Success Rate**: 99.9%+
- **Error Rate**: < 0.1%
- **Data Loss**: **0%** (backpressure test)

## Test Configuration

Modify `MAANGLoadConfig` in `maanf_load_test.go`:

```go
config := MAANGLoadConfig{
    ConcurrentUsers:  500,     // Parallel clients
    RequestsPerUser:  1000,    // Requests per client
    Duration:         30 * time.Second,
    TargetRPS:        25000,   // Target requests/sec
    CreatePercentage: 20,      // 20% creates
    ListPercentage:   50,      // 50% lists
    ReadPercentage:   30,      // 30% reads
}
```

## Interpreting Results

### Good Result Example
```
╔══════════════════════════════════════════════════════════════════╗
║  MAANG LOAD TEST: Sustained Load (15s)                          ║
╠══════════════════════════════════════════════════════════════════╣
║  THROUGHPUT                                                      ║
║    Total Requests:     225000                                    ║
║    Duration:           15s                                       ║
║    Actual RPS:         15000                                     ║
║    Peak RPS:           18500                                     ║
╠══════════════════════════════════════════════════════════════════╣
║  LATENCY (Target: P99 < 50ms)                                    ║
║    Mean:               8ms                                       ║
║    P50:                5ms          ✓                            ║
║    P95:                15ms         ✓                            ║
║    P99:                35ms         ✓                            ║
║    P99.9:              75ms                                      ║
╠══════════════════════════════════════════════════════════════════╣
║  RELIABILITY (Target: 99.9% success)                             ║
║    Success Rate:       99.9500%                                  ║
║    Error Rate:         0.0500%      ✓                            ║
╠══════════════════════════════════════════════════════════════════╣
║  RESOURCES                                                       ║
║    Memory:             512   MB                                  ║
║    Goroutines:         523                                       ║
╚══════════════════════════════════════════════════════════════════╝
```

### Red Flags 🚩
- ✗ marks next to P95/P99 latency
- Error rate > 0.1%
- Memory growth > 2x in soak test
- **Data loss > 0 in backpressure test** (CRITICAL)
- RPS significantly below target

## Backpressure Test (Critical)

The `TestMAANG_StreamingBackpressureTest` validates the MAANF reliability overhaul:

```bash
=== RUN   TestMAANG_StreamingBackpressureTest
Backpressure Test Results:
  Sent: 10000
  Acknowledged: 9985
  Failed (errors): 15    # Explicit errors (connection drops, etc.)
  Data Loss: 0 (0.0000%) # MUST BE ZERO
✓ Zero data loss confirmed under backpressure
--- PASS: TestMAANG_StreamingBackpressureTest (2.34s)
```

**Expected behavior:**
- `Sent` = `Acknowledged` + `Failed`
- `Data Loss` **MUST BE 0**
- If data loss > 0, the reliability overhaul has a bug

## Troubleshooting

### Tests Fail to Connect
```bash
# Check if server is running
curl http://localhost:8080/health
grpcurl -plaintext localhost:9090 list

# Check if ports are available
lsof -i :8080
lsof -i :9090
```

### Low RPS
- Check CPU usage (< 80% target)
- Monitor RabbitMQ queue depth
- Increase `ConcurrentUsers` or reduce workload mix

### High Error Rate
- Check server logs for errors
- Verify database connectivity
- Monitor queue broker (RabbitMQ/Kafka)

### Memory Growth (Soak Test)
- Check for goroutine leaks (`pprof`)
- Monitor connection pool leaks
- Review storage/queue adapters for missing cleanup

## Integration with CI/CD

### GitHub Actions Example
```yaml
name: Load Tests
on: [pull_request]
jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start Services
        run: docker-compose up -d
      - name: Run Load Tests
        run: go test -v ./tests/loadtest -run TestMAANG_SustainedLoad -timeout 5m
```

## Next Steps

1. Run basic tests to establish baseline
2. Run spike test to validate recovery
3. **Run backpressure test to confirm zero data loss**
4. Run soak test for memory leak detection
5. Tune `concurrency` in orchestrator if needed

## References

- [Friendship Service Tests](../../friendship-service/tests/loadtest/) - Original reference
- [MAANF Scale Guidelines](../../docs/MAANF_SCALE.md)
- [RabbitMQ Performance Tuning](https://www.rabbitmq.com/performance.html)
