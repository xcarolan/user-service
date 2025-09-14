package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := New(reg, reg)

	t.Run("record request", func(t *testing.T) {
		metrics.RecordRequest("GET", "/test", "200", time.Second)
	})

	t.Run("record request in flight", func(t *testing.T) {
		metrics.RecordRequestInFlight(1)
		metrics.RecordRequestInFlight(-1)
	})

	t.Run("set users total", func(t *testing.T) {
		metrics.SetUsersTotal(10)
	})

	t.Run("record user lookup", func(t *testing.T) {
		metrics.RecordUserLookup("found")
		metrics.RecordUserLookup("not_found")
	})

	t.Run("record error", func(t *testing.T) {
		metrics.RecordError("test_error", "/test")
	})

	t.Run("record rate limit hit", func(t *testing.T) {
		metrics.RecordRateLimitHit()
	})

	t.Run("record panic recovery", func(t *testing.T) {
		metrics.RecordPanicRecovery()
	})

	t.Run("update last request time", func(t *testing.T) {
		metrics.UpdateLastRequestTime()
	})

	t.Run("handler", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/metrics", nil)
		metrics.Handler().ServeHTTP(rr, req)

		if rr.Code != 200 {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		body := rr.Body.String()
		if !strings.Contains(body, "http_requests_total") {
			t.Errorf("expected metrics body to contain http_requests_total, got %s", body)
		}
	})
}
