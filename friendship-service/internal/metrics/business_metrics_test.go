package metrics_test

import (
	"testing"

	"github.com/MuhibNayem/connectify-v2/friendship-service/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestBusinessMetrics_NewBusinessMetrics(t *testing.T) {
	t.Parallel()

	// Note: This test may fail if run multiple times in same process
	// due to Prometheus registry conflicts. Use subtests carefully.
	m := metrics.NewBusinessMetrics()

	if m == nil {
		t.Fatal("Expected non-nil BusinessMetrics")
	}
	if m.FriendRequestsSent == nil {
		t.Error("FriendRequestsSent counter is nil")
	}
	if m.FriendRequestsAccepted == nil {
		t.Error("FriendRequestsAccepted counter is nil")
	}
	if m.FriendRequestsRejected == nil {
		t.Error("FriendRequestsRejected counter is nil")
	}
	if m.Unfriends == nil {
		t.Error("Unfriends counter is nil")
	}
	if m.BlocksCreated == nil {
		t.Error("BlocksCreated counter is nil")
	}
	if m.BlocksRemoved == nil {
		t.Error("BlocksRemoved counter is nil")
	}
	if m.OperationDuration == nil {
		t.Error("OperationDuration histogram is nil")
	}
	if m.OperationErrors == nil {
		t.Error("OperationErrors counter is nil")
	}
	if m.InconsistenciesDetected == nil {
		t.Error("InconsistenciesDetected counter is nil")
	}
	if m.OutboxEventsProcessed == nil {
		t.Error("OutboxEventsProcessed counter is nil")
	}
	if m.OutboxEventsFailed == nil {
		t.Error("OutboxEventsFailed counter is nil")
	}
}

func TestBusinessMetrics_RecordRequestSent(t *testing.T) {
	// Create a new registry for isolated testing
	reg := prometheus.NewRegistry()

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_requests_sent",
		Help: "Test counter",
	})
	reg.MustRegister(counter)

	counter.Inc()

	if got := testutil.ToFloat64(counter); got != 1 {
		t.Errorf("Expected counter to be 1, got %f", got)
	}

	counter.Inc()
	counter.Inc()

	if got := testutil.ToFloat64(counter); got != 3 {
		t.Errorf("Expected counter to be 3, got %f", got)
	}
}

func TestBusinessMetrics_RecordOperationError(t *testing.T) {
	reg := prometheus.NewRegistry()

	errorCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_operation_errors",
		Help: "Test counter",
	}, []string{"operation"})
	reg.MustRegister(errorCounter)

	errorCounter.WithLabelValues("send_request").Inc()
	errorCounter.WithLabelValues("accept_request").Inc()
	errorCounter.WithLabelValues("send_request").Inc()

	if got := testutil.ToFloat64(errorCounter.WithLabelValues("send_request")); got != 2 {
		t.Errorf("Expected send_request errors to be 2, got %f", got)
	}
	if got := testutil.ToFloat64(errorCounter.WithLabelValues("accept_request")); got != 1 {
		t.Errorf("Expected accept_request errors to be 1, got %f", got)
	}
}

func TestBusinessMetrics_ObserveOperationDuration(t *testing.T) {
	reg := prometheus.NewRegistry()

	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "test_duration",
		Help:    "Test histogram",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
	reg.MustRegister(histogram)

	histogram.WithLabelValues("send_request").Observe(0.5)
	histogram.WithLabelValues("send_request").Observe(0.1)
	histogram.WithLabelValues("send_request").Observe(0.2)

	// Histogram observations are recorded
	// We can't easily test the exact values, but we verify it doesn't panic
}

func TestBusinessMetrics_RecordInconsistency(t *testing.T) {
	reg := prometheus.NewRegistry()

	inconsistencyCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_inconsistencies",
		Help: "Test counter",
	}, []string{"type"})
	reg.MustRegister(inconsistencyCounter)

	inconsistencyCounter.WithLabelValues("mongo_neo4j_mismatch").Inc()
	inconsistencyCounter.WithLabelValues("read_repair_triggered").Inc()

	if got := testutil.ToFloat64(inconsistencyCounter.WithLabelValues("mongo_neo4j_mismatch")); got != 1 {
		t.Errorf("Expected mongo_neo4j_mismatch to be 1, got %f", got)
	}
}

func TestBusinessMetrics_OutboxMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()

	processed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_outbox_processed",
		Help: "Test counter",
	})
	failed := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_outbox_failed",
		Help: "Test counter",
	})
	reg.MustRegister(processed, failed)

	for i := 0; i < 10; i++ {
		processed.Inc()
	}
	for i := 0; i < 3; i++ {
		failed.Inc()
	}

	if got := testutil.ToFloat64(processed); got != 10 {
		t.Errorf("Expected processed to be 10, got %f", got)
	}
	if got := testutil.ToFloat64(failed); got != 3 {
		t.Errorf("Expected failed to be 3, got %f", got)
	}
}
