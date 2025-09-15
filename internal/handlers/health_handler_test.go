package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"user-service/internal/database/mocks"
	"user-service/internal/metrics"
	"user-service/internal/services"
)

func TestHealthHandler(t *testing.T) {
	// Create a mock for DBTX
	dbMock := &mocks.MockDBTX{}

	// Expect GetUsersCount to be called and return a count and no error
	mockRow := &mocks.MockRow{}
	mockRow.On("Scan", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(0).([]interface{})
		*arg[0].(*int) = 5 // Mock a count of 5 users
	})
	dbMock.On("QueryRow", context.Background(), "SELECT COUNT(*) FROM users", mock.Anything).Return(mockRow)

	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)
	userService := services.NewUserService(dbMock, metricsCollector)
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

	// Assert that the mock expectations were met
	dbMock.AssertExpectations(t)
}
