package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics holds Prometheus metrics for friendship operations
type BusinessMetrics struct {
	FriendRequestsSent     prometheus.Counter
	FriendRequestsAccepted prometheus.Counter
	FriendRequestsRejected prometheus.Counter
	Unfriends              prometheus.Counter
	BlocksCreated          prometheus.Counter
	BlocksRemoved          prometheus.Counter

	OperationDuration *prometheus.HistogramVec
	OperationErrors   *prometheus.CounterVec
}

// NewBusinessMetrics creates and registers business metrics
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		FriendRequestsSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_requests_sent_total",
			Help: "Total number of friend requests sent",
		}),
		FriendRequestsAccepted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_requests_accepted_total",
			Help: "Total number of friend requests accepted",
		}),
		FriendRequestsRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_requests_rejected_total",
			Help: "Total number of friend requests rejected",
		}),
		Unfriends: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_unfriends_total",
			Help: "Total number of unfriend actions",
		}),
		BlocksCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_blocks_created_total",
			Help: "Total number of user blocks",
		}),
		BlocksRemoved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "friendship_blocks_removed_total",
			Help: "Total number of user unblocks",
		}),
		OperationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "friendship_operation_duration_seconds",
			Help:    "Duration of friendship operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		OperationErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "friendship_operation_errors_total",
			Help: "Total number of errors by operation",
		}, []string{"operation"}),
	}
}

// RecordRequestSent increments the friend request sent counter
func (m *BusinessMetrics) RecordRequestSent() {
	m.FriendRequestsSent.Inc()
}

// RecordRequestAccepted increments the friend request accepted counter
func (m *BusinessMetrics) RecordRequestAccepted() {
	m.FriendRequestsAccepted.Inc()
}

// RecordRequestRejected increments the friend request rejected counter
func (m *BusinessMetrics) RecordRequestRejected() {
	m.FriendRequestsRejected.Inc()
}

// RecordUnfriend increments the unfriend counter
func (m *BusinessMetrics) RecordUnfriend() {
	m.Unfriends.Inc()
}

// RecordBlock increments the block counter
func (m *BusinessMetrics) RecordBlock() {
	m.BlocksCreated.Inc()
}

// RecordUnblock increments the unblock counter
func (m *BusinessMetrics) RecordUnblock() {
	m.BlocksRemoved.Inc()
}

// RecordOperationError increments the error counter for an operation
func (m *BusinessMetrics) RecordOperationError(operation string) {
	m.OperationErrors.WithLabelValues(operation).Inc()
}

// ObserveOperationDuration records the duration of an operation
func (m *BusinessMetrics) ObserveOperationDuration(operation string, durationSeconds float64) {
	m.OperationDuration.WithLabelValues(operation).Observe(durationSeconds)
}
