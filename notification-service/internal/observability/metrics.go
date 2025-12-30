package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics provides SLO-grade observability for the notification service
var Metrics = newMetrics()

type metrics struct {
	// Notification lifecycle
	NotificationsCreated   *prometheus.CounterVec
	NotificationsDelivered *prometheus.CounterVec
	NotificationsFailed    *prometheus.CounterVec

	// Delivery latency (p50, p95, p99)
	DeliveryLatency *prometheus.HistogramVec

	// Channel-specific metrics
	ChannelAttempts  *prometheus.CounterVec
	ChannelSuccesses *prometheus.CounterVec
	ChannelFailures  *prometheus.CounterVec
	ChannelLatency   *prometheus.HistogramVec
	ChannelRetries   *prometheus.CounterVec

	// Queue metrics
	QueueDepth         *prometheus.GaugeVec
	QueueLatency       *prometheus.HistogramVec
	BackpressureEvents prometheus.Counter

	// DLQ metrics
	DLQEvents      prometheus.Counter
	DLQDepth       prometheus.Gauge
	DLQReprocessed prometheus.Counter

	// Idempotency
	DuplicatesSkipped prometheus.Counter
	IdempotencyErrors prometheus.Counter

	// Circuit breaker
	CircuitBreakerTrips *prometheus.CounterVec
	CircuitBreakerState *prometheus.GaugeVec

	// WebSocket connections
	WebSocketConnections  prometheus.Gauge
	WebSocketMessages     prometheus.Counter
	WebSocketBackpressure prometheus.Counter

	// Rate limiting
	RateLimitRejections *prometheus.CounterVec
}

func newMetrics() *metrics {
	return &metrics{
		// Notification lifecycle
		NotificationsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_created_total",
				Help: "Total notifications created",
			},
			[]string{"type", "priority"},
		),
		NotificationsDelivered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_delivered_total",
				Help: "Total notifications successfully delivered",
			},
			[]string{"type", "channel"},
		),
		NotificationsFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_failed_total",
				Help: "Total notifications that failed delivery",
			},
			[]string{"type", "channel", "error_type"},
		),

		// Delivery latency with SLO buckets
		DeliveryLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notification_delivery_latency_seconds",
				Help:    "End-to-end delivery latency from creation to delivery",
				Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"channel", "priority"},
		),

		// Channel-specific metrics
		ChannelAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "channel_delivery_attempts_total",
				Help: "Total delivery attempts per channel",
			},
			[]string{"channel"},
		),
		ChannelSuccesses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "channel_delivery_success_total",
				Help: "Successful deliveries per channel",
			},
			[]string{"channel"},
		),
		ChannelFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "channel_delivery_failure_total",
				Help: "Failed deliveries per channel",
			},
			[]string{"channel", "error_type"},
		),
		ChannelLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "channel_delivery_latency_seconds",
				Help:    "Per-channel delivery latency",
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
			[]string{"channel"},
		),
		ChannelRetries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "channel_delivery_retries_total",
				Help: "Total retries per channel",
			},
			[]string{"channel"},
		),

		// Queue metrics
		QueueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_depth",
				Help: "Current queue depth",
			},
			[]string{"queue", "priority"},
		),
		QueueLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_processing_latency_seconds",
				Help:    "Time from queue arrival to processing start",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
			},
			[]string{"queue"},
		),
		BackpressureEvents: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "backpressure_events_total",
				Help: "Total backpressure events (queue saturation)",
			},
		),

		// DLQ metrics
		DLQEvents: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "dlq_events_total",
				Help: "Total events sent to DLQ",
			},
		),
		DLQDepth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dlq_depth",
				Help: "Current DLQ depth",
			},
		),
		DLQReprocessed: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "dlq_reprocessed_total",
				Help: "Total DLQ events reprocessed",
			},
		),

		// Idempotency
		DuplicatesSkipped: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "idempotency_duplicates_skipped_total",
				Help: "Total duplicate events skipped",
			},
		),
		IdempotencyErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "idempotency_errors_total",
				Help: "Total idempotency check errors",
			},
		),

		// Circuit breaker
		CircuitBreakerTrips: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "circuit_breaker_trips_total",
				Help: "Total circuit breaker trips",
			},
			[]string{"channel"},
		),
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"channel"},
		),

		// WebSocket
		WebSocketConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "websocket_connections_active",
				Help: "Current active WebSocket connections",
			},
		),
		WebSocketMessages: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "websocket_messages_sent_total",
				Help: "Total WebSocket messages sent",
			},
		),
		WebSocketBackpressure: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "websocket_backpressure_total",
				Help: "Total WebSocket backpressure events",
			},
		),

		// Rate limiting
		RateLimitRejections: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limit_rejections_total",
				Help: "Total requests rejected by rate limiter",
			},
			[]string{"endpoint"},
		),
	}
}

// RecordDelivery records a delivery attempt with timing
func (m *metrics) RecordDelivery(channel, priority string, success bool, latency time.Duration, errType string) {
	m.ChannelAttempts.WithLabelValues(channel).Inc()
	m.ChannelLatency.WithLabelValues(channel).Observe(latency.Seconds())

	if success {
		m.ChannelSuccesses.WithLabelValues(channel).Inc()
		m.DeliveryLatency.WithLabelValues(channel, priority).Observe(latency.Seconds())
	} else {
		m.ChannelFailures.WithLabelValues(channel, errType).Inc()
	}
}

// RecordRetry records a delivery retry
func (m *metrics) RecordRetry(channel string) {
	m.ChannelRetries.WithLabelValues(channel).Inc()
}

// RecordDLQ records an event sent to DLQ
func (m *metrics) RecordDLQ() {
	m.DLQEvents.Inc()
}

// RecordDuplicate records a skipped duplicate
func (m *metrics) RecordDuplicate() {
	m.DuplicatesSkipped.Inc()
}

// RecordBackpressure records a backpressure event
func (m *metrics) RecordBackpressure() {
	m.BackpressureEvents.Inc()
}

// UpdateQueueDepth updates queue depth gauge
func (m *metrics) UpdateQueueDepth(queue, priority string, depth int) {
	m.QueueDepth.WithLabelValues(queue, priority).Set(float64(depth))
}

// UpdateWebSocketConnections updates active connection count
func (m *metrics) UpdateWebSocketConnections(count int) {
	m.WebSocketConnections.Set(float64(count))
}
