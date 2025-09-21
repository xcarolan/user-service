package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"user-service/internal/database/mocks"
	"user-service/internal/metrics"
	"user-service/internal/services"
)

func TestUserHandler(t *testing.T) {
	reg := prometheus.NewRegistry()
	metricsCollector := metrics.New(reg, reg)

	t.Run("get user", func(t *testing.T) {
		// Create a mock for DBTX
		dbMock := &mocks.MockDBTX{}

		// Setup expectations for GetUser
		row := &mocks.MockRow{}
		row.On("Scan", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(0).([]interface{})
			*arg[0].(*int) = 1
			*arg[1].(*string) = "John Doe"
			*arg[2].(*string) = "john@example.com"
		})
		dbMock.On("QueryRow", context.Background(), "SELECT id, name, email FROM users WHERE id = $1", 1).Return(row)

		userService := services.NewUserService(dbMock, metricsCollector)
		userHandler := NewUserHandler(userService)
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
		dbMock.AssertExpectations(t)
	})

	t.Run("get user table driven", func(t *testing.T) {
		// Create a mock for DBTX
		dbMock := &mocks.MockDBTX{}

		// Setup expectations for GetUser (non-existent)
		notFoundRow := &mocks.MockRow{}
		notFoundRow.On("Scan", mock.Anything).Return(pgx.ErrNoRows)
		dbMock.On("QueryRow", context.Background(), "SELECT id, name, email FROM users WHERE id = $1", 100).Return(notFoundRow)

		userService := services.NewUserService(dbMock, metricsCollector)
		userHandler := NewUserHandler(userService)

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
		dbMock.AssertExpectations(t)
	})

	t.Run("list users", func(t *testing.T) {
		// Create a mock for DBTX
		dbMock := &mocks.MockDBTX{}

		// Setup expectations for ListUsers
		rows := &mocks.MockRows{}
		rows.On("Close").Return()
		rows.On("Next").Return(true).Once()
		rows.On("Next").Return(false).Once()
		rows.On("Scan", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(0).([]interface{})
			*arg[0].(*int) = 1
			*arg[1].(*string) = "John Doe"
			*arg[2].(*string) = "john@example.com"
		})
		dbMock.On("Query", context.Background(), "SELECT id, name, email FROM users").Return(rows, nil)

		userService := services.NewUserService(dbMock, metricsCollector)
		userHandler := NewUserHandler(userService)

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
		dbMock.AssertExpectations(t)
	})

	t.Run("list users database error", func(t *testing.T) {
		// Create a mock for DBTX
		dbMock := &mocks.MockDBTX{}

		// Setup expectations for database error
		dbMock.On("Query", context.Background(), "SELECT id, name, email FROM users").Return(nil, errors.New("database error"))

		userService := services.NewUserService(dbMock, metricsCollector)
		userHandler := NewUserHandler(userService)

		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h := http.HandlerFunc(userHandler.ListUsers)

		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusInternalServerError)
		}
		dbMock.AssertExpectations(t)
	})
}
