package config

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metrics
type Metrics struct{}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// MetricsHandler returns the Prometheus metrics HTTP handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
