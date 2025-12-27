package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BusinessMetrics struct {
	StoriesCreated    prometheus.Counter
	StoriesViewed     prometheus.Counter
	StoriesDeleted    prometheus.Counter
	ReactionsAdded    prometheus.Counter
	ViewersAccessed   prometheus.Counter
	FeedRequests      prometheus.Counter
	RateLimitHits     *prometheus.CounterVec
}

func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		StoriesCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "stories_created_total",
			Help: "Total number of stories created",
		}),
		StoriesViewed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "stories_viewed_total",
			Help: "Total number of story views",
		}),
		StoriesDeleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "stories_deleted_total",
			Help: "Total number of stories deleted",
		}),
		ReactionsAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "story_reactions_total",
			Help: "Total number of story reactions",
		}),
		ViewersAccessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "story_viewers_accessed_total",
			Help: "Total number of story viewer list accesses",
		}),
		FeedRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "story_feed_requests_total",
			Help: "Total number of story feed requests",
		}),
		RateLimitHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "story_service_rate_limit_hits_total",
			Help: "Number of rate-limited requests grouped by action",
		}, []string{"action"}),
	}
}

func (m *BusinessMetrics) IncrementStoriesCreated() {
	if m != nil {
		m.StoriesCreated.Inc()
	}
}

func (m *BusinessMetrics) IncrementStoriesViewed() {
	if m != nil {
		m.StoriesViewed.Inc()
	}
}

func (m *BusinessMetrics) IncrementStoriesDeleted() {
	if m != nil {
		m.StoriesDeleted.Inc()
	}
}

func (m *BusinessMetrics) IncrementReactions() {
	if m != nil {
		m.ReactionsAdded.Inc()
	}
}

func (m *BusinessMetrics) IncrementViewersAccessed() {
	if m != nil {
		m.ViewersAccessed.Inc()
	}
}

func (m *BusinessMetrics) IncrementFeedRequests() {
	if m != nil {
		m.FeedRequests.Inc()
	}
}

func (m *BusinessMetrics) RecordRateLimitHit(action string) {
	if m == nil || action == "" {
		return
	}
	m.RateLimitHits.WithLabelValues(action).Inc()
}