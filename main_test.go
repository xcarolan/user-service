// main_test.go - Updated unit tests for metrics version
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Test data
var testUsers = map[int]User{
	1: {ID: 1, Name: "John Doe", Email: "john@example.com"},
	2: {ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
}

// Helper function to create a test service with metrics
func createTestService() (*UserService, *Metrics) {
	// Create a new registry for tests to avoid conflicts
	registry := prometheus.NewRegistry()

	// Create metrics with custom registry
	metrics := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "http_requests_total_test"},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "http_request_duration_seconds_test"},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "http_requests_in_flight_test"},
		),
		usersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "users_total_test"},
		),
		userLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "user_lookups_total_test"},
			[]string{"result"},
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "errors_total_test"},
			[]string{"type", "endpoint"},
		),
		rateLimitHits: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "rate_limit_hits_total_test"},
		),
		panicRecoveries: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "panic_recoveries_total_test"},
		),
		lastRequestTime: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "last_request_time_seconds_test"},
		),
		uptime: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "uptime_seconds_total_test"},
		),
	}

	// Register with test registry
	registry.MustRegister(
		metrics.requestsTotal,
		metrics.requestDuration,
		metrics.requestsInFlight,
		metrics.usersTotal,
		metrics.userLookups,
		metrics.errorRate,
		metrics.rateLimitHits,
		metrics.panicRecoveries,
		metrics.lastRequestTime,
		metrics.uptime,
	)

	service := &UserService{
		users:   testUsers,
		metrics: metrics,
	}

	// Initialize metrics
	service.metrics.usersTotal.Set(float64(len(service.users)))

	return service, metrics
}

// Helper function to perform HTTP request and return response
func performRequest(handler http.HandlerFunc, method, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestUserService_GetUser_Success(t *testing.T) {
	service, _ := createTestService()

	tests := []struct {
		name     string
		userID   string
		expected User
	}{
		{
			name:     "Get user 1",
			userID:   "1",
			expected: User{ID: 1, Name: "John Doe", Email: "john@example.com"},
		},
		{
			name:     "Get user 2",
			userID:   "2",
			expected: User{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := performRequest(service.GetUser, "GET", "/user?id="+tt.userID)

			// Check status code
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Parse and check response body
			var user User
			if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if user != tt.expected {
				t.Errorf("Expected user %+v, got %+v", tt.expected, user)
			}
		})
	}
}

func TestUserService_GetUser_MissingID(t *testing.T) {
	service, _ := createTestService()
	rr := performRequest(service.GetUser, "GET", "/user")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	expectedError := "missing id parameter"
	if !strings.Contains(rr.Body.String(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got '%s'",
			expectedError, rr.Body.String())
	}
}

func TestUserService_GetUser_InvalidID(t *testing.T) {
	service, _ := createTestService()

	tests := []struct {
		name      string
		invalidID string
	}{
		{"Non-numeric ID", "abc"},
		{"Floating point ID", "1.5"},
		{"Special characters", "!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/user?id=" + tt.invalidID
			rr := performRequest(service.GetUser, "GET", url)

			expectedStatus := http.StatusBadRequest
			if rr.Code != expectedStatus {
				t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
			}
		})
	}
}

func TestUserService_GetUser_NotFound(t *testing.T) {
	service, _ := createTestService()
	rr := performRequest(service.GetUser, "GET", "/user?id=999")

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	expectedError := "user not found"
	if !strings.Contains(rr.Body.String(), expectedError) {
		t.Errorf("Expected error message to contain '%s', got '%s'",
			expectedError, rr.Body.String())
	}
}

func TestUserService_ListUsers_Success(t *testing.T) {
	service, _ := createTestService()
	rr := performRequest(service.ListUsers, "GET", "/users")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check total count
	total, ok := response["total"].(float64) // JSON numbers are float64
	if !ok {
		t.Fatal("Expected 'total' field in response")
	}

	expectedTotal := float64(len(testUsers))
	if total != expectedTotal {
		t.Errorf("Expected total %v, got %v", expectedTotal, total)
	}

	// Check users array exists
	if _, ok := response["users"]; !ok {
		t.Error("Expected 'users' field in response")
	}
}

func TestUserService_HealthCheck(t *testing.T) {
	service, _ := createTestService()
	rr := performRequest(service.HealthCheck, "GET", "/health")

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check required fields
	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if service, ok := response["service"].(string); !ok || service != "user-service" {
		t.Errorf("Expected service 'user-service', got %v", response["service"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected 'timestamp' field in response")
	}
}

// Test concurrent access to ensure thread safety
func TestUserService_ConcurrentAccess(t *testing.T) {
	service, _ := createTestService()

	// Number of concurrent goroutines
	numGoroutines := 100
	done := make(chan bool, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			// Alternate between different user IDs
			userID := (goroutineID % 2) + 1
			url := "/user?id=" + string(rune(userID+'0'))

			rr := performRequest(service.GetUser, "GET", url)

			if rr.Code != http.StatusOK {
				t.Errorf("Concurrent request failed with status %d", rr.Code)
				return
			}

			var user User
			if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
				t.Errorf("Failed to unmarshal concurrent response: %v", err)
				return
			}

			if user.ID != userID {
				t.Errorf("Expected user ID %d, got %d", userID, user.ID)
			}
		}(i)
	}

	// Wait for all goroutines to complete with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case <-timeout:
			t.Fatal("Concurrent test timed out")
		}
	}
}

// Benchmark tests
func BenchmarkUserService_GetUser(b *testing.B) {
	service, _ := createTestService()
	req := httptest.NewRequest("GET", "/user?id=1", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		service.GetUser(rr, req)
	}
}

func BenchmarkUserService_ListUsers(b *testing.B) {
	service, _ := createTestService()
	req := httptest.NewRequest("GET", "/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		service.ListUsers(rr, req)
	}
}

// Test middleware functionality
func TestMiddleware_Logging(t *testing.T) {
	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply logging middleware
	wrappedHandler := loggingMiddleware(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if body := rr.Body.String(); body != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", body)
	}
}

func TestMiddleware_Recovery(t *testing.T) {
	// Create test metrics
	_, metrics := createTestService()

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Apply recovery middleware with metrics
	wrappedHandler := recoveryMiddleware(metrics)(handler)

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Should recover and return 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestMiddleware_CORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := corsMiddleware(handler)

	// Test OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	for header, expectedValue := range expectedHeaders {
		if actualValue := rr.Header().Get(header); actualValue != expectedValue {
			t.Errorf("Expected header %s: %s, got: %s", header, expectedValue, actualValue)
		}
	}

	// OPTIONS should return 200
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusOK, rr.Code)
	}
}

// Table-driven test for comprehensive input validation
func TestUserService_GetUser_InputValidation(t *testing.T) {
	service, _ := createTestService()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid ID",
			url:            "/user?id=1",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "Missing ID parameter",
			url:            "/user",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing id parameter",
		},
		{
			name:           "Empty ID parameter",
			url:            "/user?id=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing id parameter",
		},
		{
			name:           "Non-numeric ID",
			url:            "/user?id=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id parameter",
		},
		{
			name:           "Negative ID",
			url:            "/user?id=-1",
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
		{
			name:           "Zero ID",
			url:            "/user?id=0",
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
		{
			name:           "Large ID",
			url:            "/user?id=999999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := performRequest(service.GetUser, "GET", tt.url)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError != "" {
				if !strings.Contains(rr.Body.String(), tt.expectedError) {
					t.Errorf("Expected error message to contain '%s', got '%s'",
						tt.expectedError, rr.Body.String())
				}
			}
		})
	}
}

// Test metrics functionality
func TestMetrics_Collection(t *testing.T) {
	service, metrics := createTestService()

	// Make a successful request
	rr := performRequest(service.GetUser, "GET", "/user?id=1")
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected successful request, got status %d", rr.Code)
	}

	// Check that metrics were updated
	// Note: In a real test, you'd gather metrics from the registry and check values
	// This is a basic structure test
	if metrics.userLookups == nil {
		t.Error("Expected userLookups metric to be initialized")
	}

	if metrics.errorRate == nil {
		t.Error("Expected errorRate metric to be initialized")
	}
}
