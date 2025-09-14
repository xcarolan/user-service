package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"user-service/internal/metrics"
	"user-service/internal/services"
)

func TestHealthHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	userService := services.NewUserService(metricsCollector)
	healthHandler := NewHealthHandler(userService)

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(healthHandler.Health)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
