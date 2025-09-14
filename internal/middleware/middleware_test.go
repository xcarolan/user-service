package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	"user-service/internal/metrics"
)

func TestLogging(t *testing.T) {
	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply logging middleware
	wrappedHandler := Logging()(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply metrics middleware
	wrappedHandler := Metrics(metricsCollector)(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestRateLimit(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	limiter := rate.NewLimiter(1, 1)
	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply rate limit middleware
	wrappedHandler := RateLimit(limiter, metricsCollector)(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Make another request to trigger rate limit
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
}

func TestCORS(t *testing.T) {
	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply CORS middleware
	wrappedHandler := CORS()(handler)

	// Make request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestRecovery(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Apply recovery middleware
	wrappedHandler := Recovery(metricsCollector)(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
