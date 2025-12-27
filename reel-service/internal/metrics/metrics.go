package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BusinessMetrics struct {
	ReelsCreated   prometheus.Counter
	ReelsDeleted   prometheus.Counter
	ReelsViewed    prometheus.Counter
	ReactionsAdded prometheus.Counter
	CommentsAdded  prometheus.Counter
}

func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		ReelsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "reel_service_reels_created_total",
			Help: "Total number of reels created",
		}),
		ReelsDeleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "reel_service_reels_deleted_total",
			Help: "Total number of reels deleted",
		}),
		ReelsViewed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "reel_service_reels_viewed_total",
			Help: "Total number of reel views",
		}),
		ReactionsAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "reel_service_reactions_added_total",
			Help: "Total number of reactions added",
		}),
		CommentsAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "reel_service_comments_added_total",
			Help: "Total number of comments added",
		}),
	}
}
