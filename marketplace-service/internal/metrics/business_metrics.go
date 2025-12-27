package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BusinessMetrics struct {
	ProductsCreated prometheus.Counter
	ProductsDeleted prometheus.Counter
	ProductsSold    prometheus.Counter
	ProductViews    prometheus.Counter
	RateLimitHits   *prometheus.CounterVec
}

var (
	productsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_created_total",
		Help: "Total number of products created",
	})
	productsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_deleted_total",
		Help: "Total number of products deleted",
	})
	productsSold = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_sold_total",
		Help: "Total number of products marked as sold",
	})
	productViews = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_views_total",
		Help: "Total number of product views",
	})
	rateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "marketplace_rate_limit_hits_total",
		Help: "Number of rate-limited requests grouped by action",
	}, []string{"action"})
)

func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		ProductsCreated: productsCreated,
		ProductsDeleted: productsDeleted,
		ProductsSold:    productsSold,
		ProductViews:    productViews,
		RateLimitHits:   rateLimitHits,
	}
}

func (m *BusinessMetrics) IncrementProductsCreated() {
	if m != nil {
		m.ProductsCreated.Inc()
	}
}

func (m *BusinessMetrics) IncrementProductsDeleted() {
	if m != nil {
		m.ProductsDeleted.Inc()
	}
}

func (m *BusinessMetrics) IncrementProductsSold() {
	if m != nil {
		m.ProductsSold.Inc()
	}
}

func (m *BusinessMetrics) IncrementProductViews() {
	if m != nil {
		m.ProductViews.Inc()
	}
}

func (m *BusinessMetrics) RecordRateLimitHit(action string) {
	if m != nil {
		m.RateLimitHits.WithLabelValues(action).Inc()
	}
}
