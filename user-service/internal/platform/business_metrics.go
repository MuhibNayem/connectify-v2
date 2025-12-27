package platform

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics holds Prometheus counters for user-service KPIs
type BusinessMetrics struct {
	UsersRegistered      prometheus.Counter
	LoginAttempts        *prometheus.CounterVec
	ProfileUpdates       prometheus.Counter
	PasswordChanges      prometheus.Counter
	EmailChanges         prometheus.Counter
	TwoFactorToggles     *prometheus.CounterVec
	AccountDeactivations prometheus.Counter
	RateLimitHits        *prometheus.CounterVec
}

// NewBusinessMetrics creates and registers business metrics
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		UsersRegistered: promauto.NewCounter(prometheus.CounterOpts{
			Name: "users_registered_total",
			Help: "Total number of users registered",
		}),
		LoginAttempts: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "user_login_attempts_total",
			Help: "Total number of login attempts by status",
		}, []string{"status"}), // success, failed, locked
		ProfileUpdates: promauto.NewCounter(prometheus.CounterOpts{
			Name: "user_profile_updates_total",
			Help: "Total number of profile updates",
		}),
		PasswordChanges: promauto.NewCounter(prometheus.CounterOpts{
			Name: "user_password_changes_total",
			Help: "Total number of password changes",
		}),
		EmailChanges: promauto.NewCounter(prometheus.CounterOpts{
			Name: "user_email_changes_total",
			Help: "Total number of email changes",
		}),
		TwoFactorToggles: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "user_2fa_toggles_total",
			Help: "Total number of 2FA toggles",
		}, []string{"action"}), // enabled, disabled
		AccountDeactivations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "user_account_deactivations_total",
			Help: "Total number of account deactivations",
		}),
		RateLimitHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "user_service_rate_limit_hits_total",
			Help: "Number of rate-limited requests grouped by action",
		}, []string{"action"}),
	}
}

// IncrementRegistrations increments the registrations counter
func (m *BusinessMetrics) IncrementRegistrations() {
	m.UsersRegistered.Inc()
}

// IncrementLogin increments the login counter
func (m *BusinessMetrics) IncrementLogin(status string) {
	m.LoginAttempts.WithLabelValues(status).Inc()
}

// IncrementProfileUpdates increments profile updates counter
func (m *BusinessMetrics) IncrementProfileUpdates() {
	m.ProfileUpdates.Inc()
}

// IncrementPasswordChanges increments password changes counter
func (m *BusinessMetrics) IncrementPasswordChanges() {
	m.PasswordChanges.Inc()
}

// IncrementEmailChanges increments email changes counter
func (m *BusinessMetrics) IncrementEmailChanges() {
	m.EmailChanges.Inc()
}

// IncrementTwoFactor increments 2FA toggle counter
func (m *BusinessMetrics) IncrementTwoFactor(action string) {
	m.TwoFactorToggles.WithLabelValues(action).Inc()
}

// IncrementDeactivations increments account deactivations counter
func (m *BusinessMetrics) IncrementDeactivations() {
	m.AccountDeactivations.Inc()
}

// RecordRateLimitHit records a rate limit hit for observability.
func (m *BusinessMetrics) RecordRateLimitHit(action string) {
	if action == "" || m == nil {
		return
	}
	m.RateLimitHits.WithLabelValues(action).Inc()
}
