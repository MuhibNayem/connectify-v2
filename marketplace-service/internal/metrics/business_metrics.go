package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics holds Prometheus counters for business KPIs.
type BusinessMetrics struct {
	ProductsCreated prometheus.Counter
	ProductsDeleted prometheus.Counter
	ProductsSold    prometheus.Counter
	ProductViews    prometheus.Counter
	RateLimitHits   *prometheus.CounterVec
}

// NewBusinessMetrics creates and registers business metrics.
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		ProductsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "products_created_total",
			Help: "Total number of products created",
		}),
		ProductsDeleted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "products_deleted_total",
			Help: "Total number of products deleted",
		}),
		ProductsSold: promauto.NewCounter(prometheus.CounterOpts{
			Name: "products_sold_total",
			Help: "Total number of products marked as sold",
		}),
		ProductViews: promauto.NewCounter(prometheus.CounterOpts{
			Name: "products_views_total",
			Help: "Total number of product views",
		}),
		RateLimitHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "marketplace_rate_limit_hits_total",
			Help: "Number of rate-limited requests grouped by action",
		}, []string{"action"}),
	}
}

// IncrementProductsCreated increments the products created counter.
func (m *BusinessMetrics) IncrementProductsCreated() {
	if m != nil {
		m.ProductsCreated.Inc()
	}
}

// IncrementProductsDeleted increments the products deleted counter.
func (m *BusinessMetrics) IncrementProductsDeleted() {
	if m != nil {
		m.ProductsDeleted.Inc()
	}
}

// IncrementProductsSold increments the products sold counter.
func (m *BusinessMetrics) IncrementProductsSold() {
	if m != nil {
		m.ProductsSold.Inc()
	}
}

// IncrementProductViews increments the product views counter.
func (m *BusinessMetrics) IncrementProductViews() {
	if m != nil {
		m.ProductViews.Inc()
	}
}

// RecordRateLimitHit increments the rate limit metric for a given action.
func (m *BusinessMetrics) RecordRateLimitHit(action string) {
	if m != nil {
		m.RateLimitHits.WithLabelValues(action).Inc()
	}
}
