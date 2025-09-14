package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"user-service/internal/metrics"
	"user-service/internal/services"
)

func TestUserHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	userService := services.NewUserService(metricsCollector)
	userHandler := NewUserHandler(userService)

	t.Run("get user", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/user?id=1", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h := http.HandlerFunc(userHandler.GetUser)

		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})

	t.Run("get user table driven", func(t *testing.T) {
		tests := []struct {
			name       string
			url        string
			wantStatus int
		}{
			{"missing id", "/user", http.StatusBadRequest},
			{"invalid id", "/user?id=abc", http.StatusBadRequest},
			{"not found id", "/user?id=100", http.StatusNotFound},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequest("GET", tt.url, nil)
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				h := http.HandlerFunc(userHandler.GetUser)

				h.ServeHTTP(rr, req)

				if status := rr.Code; status != tt.wantStatus {
					t.Errorf("handler returned wrong status code: got %v want %v",
						status, tt.wantStatus)
				}
			})
		}
	})

	t.Run("list users", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h := http.HandlerFunc(userHandler.ListUsers)

		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})
}
