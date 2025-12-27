package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics holds Prometheus counters for business KPIs.
type BusinessMetrics struct {
	EventsCreated      prometheus.Counter
	EventsDeleted      prometheus.Counter
	RSVPTotal          *prometheus.CounterVec
	InvitationsSent    prometheus.Counter
	RecommendationReqs prometheus.Counter
	PostsCreated       prometheus.Counter
}

// NewBusinessMetrics creates and registers business metrics.
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		EventsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_created_total",
			Help: "Total number of events created",
		}),
		EventsDeleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "events_deleted_total",
			Help: "Total number of events deleted",
		}),
		RSVPTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_rsvp_total",
			Help: "Total number of RSVPs by status",
		}, []string{"status"}),
		InvitationsSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "event_invitations_sent_total",
			Help: "Total number of event invitations sent",
		}),
		RecommendationReqs: promauto.NewCounter(prometheus.CounterOpts{
			Name: "event_recommendation_requests_total",
			Help: "Total number of recommendation requests",
		}),
		PostsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "event_posts_created_total",
			Help: "Total number of event discussion posts created",
		}),
	}
}

// IncrementEventsCreated increments the events created counter.
func (m *BusinessMetrics) IncrementEventsCreated() {
	if m != nil {
		m.EventsCreated.Inc()
	}
}

// IncrementEventsDeleted increments the events deleted counter.
func (m *BusinessMetrics) IncrementEventsDeleted() {
	if m != nil {
		m.EventsDeleted.Inc()
	}
}

// IncrementRSVP increments the RSVP counter for a given status.
func (m *BusinessMetrics) IncrementRSVP(status string) {
	if m != nil {
		m.RSVPTotal.WithLabelValues(status).Inc()
	}
}

// IncrementInvitations increments the invitations sent counter.
func (m *BusinessMetrics) IncrementInvitations(count int) {
	if m == nil || count <= 0 {
		return
	}
	for i := 0; i < count; i++ {
		m.InvitationsSent.Inc()
	}
}

// IncrementRecommendations increments the recommendations counter.
func (m *BusinessMetrics) IncrementRecommendations() {
	if m != nil {
		m.RecommendationReqs.Inc()
	}
}

// IncrementPosts increments the posts created counter.
func (m *BusinessMetrics) IncrementPosts() {
	if m != nil {
		m.PostsCreated.Inc()
	}
}
