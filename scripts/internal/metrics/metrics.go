package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics structure to hold all Prometheus metrics
type Metrics struct {
	// HTTP request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge

	// Business metrics
	usersTotal  prometheus.Gauge
	userLookups *prometheus.CounterVec
	errorRate   *prometheus.CounterVec

	// System metrics
	rateLimitHits   prometheus.Counter
	panicRecoveries prometheus.Counter

	// Custom application metrics
	lastRequestTime prometheus.Gauge
	uptime          prometheus.Counter
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		usersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "users_total",
				Help: "Total number of users in the system",
			},
		),
		userLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_lookups_total",
				Help: "Total number of user lookup operations",
			},
			[]string{"result"},
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "errors_total",
				Help: "Total number of errors by type",
			},
			[]string{"type", "endpoint"},
		),
		rateLimitHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "rate_limit_hits_total",
				Help: "Total number of rate limit violations",
			},
		),
		panicRecoveries: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "panic_recoveries_total",
				Help: "Total number of panic recoveries",
			},
		),
		lastRequestTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "last_request_time_seconds",
				Help: "Unix timestamp of the last request",
			},
		),
		uptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "uptime_seconds_total",
				Help: "Total uptime in seconds",
			},
		),
	}

	// Register all metrics with Prometheus
	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
		m.usersTotal,
		m.userLookups,
		m.errorRate,
		m.rateLimitHits,
		m.panicRecoveries,
		m.lastRequestTime,
		m.uptime,
	)

	// Start uptime counter
	go m.updateUptime()

	return m
}

// Handler returns the Prometheus metrics handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// RecordRequest records HTTP request metrics
func (m *Metrics) RecordRequest(method, endpoint, statusCode string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordRequestInFlight tracks requests currently being processed
func (m *Metrics) RecordRequestInFlight(delta float64) {
	m.requestsInFlight.Add(delta)
}

// SetUsersTotal sets the current users total
func (m *Metrics) SetUsersTotal(count float64) {
	m.usersTotal.Set(count)
}

// RecordUserLookup records user lookup results
func (m *Metrics) RecordUserLookup(result string) {
	m.userLookups.WithLabelValues(result).Inc()
}

// RecordError records application errors
func (m *Metrics) RecordError(errorType, endpoint string) {
	m.errorRate.WithLabelValues(errorType, endpoint).Inc()
}

// RecordRateLimitHit records rate limit violations
func (m *Metrics) RecordRateLimitHit() {
	m.rateLimitHits.Inc()
}

// RecordPanicRecovery records panic recoveries
func (m *Metrics) RecordPanicRecovery() {
	m.panicRecoveries.Inc()
}

// UpdateLastRequestTime updates the last request timestamp
func (m *Metrics) UpdateLastRequestTime() {
	m.lastRequestTime.SetToCurrentTime()
}

// Update uptime counter every second
func (m *Metrics) updateUptime() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.uptime.Inc()
	}
}
