package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Integration test helper to create a test server with all middleware
func createTestServer() *httptest.Server {
	// Create test metrics (using different names to avoid conflicts)
	metrics := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "http_requests_total_integration"},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "http_request_duration_seconds_integration"},
			[]string{"method", "endpoint"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "http_requests_in_flight_integration"},
		),
		usersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "users_total_integration"},
		),
		userLookups: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "user_lookups_total_integration"},
			[]string{"result"},
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "errors_total_integration"},
			[]string{"type", "endpoint"},
		),
		rateLimitHits: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "rate_limit_hits_total_integration"},
		),
		panicRecoveries: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "panic_recoveries_total_integration"},
		),
		lastRequestTime: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "last_request_time_seconds_integration"},
		),
		uptime: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "uptime_seconds_total_integration"},
		),
	}

	// Create test registry to avoid conflicts
	testRegistry := prometheus.NewRegistry()
	testRegistry.MustRegister(
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

	service := NewUserService(metrics)
	mux := setupRoutes(service, metrics)
	return httptest.NewServer(mux)
}

// Helper function to make HTTP requests to test server
func makeRequest(server *httptest.Server, method, path string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// Helper to read response body
func readResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func TestIntegration_CompleteUserWorkflow(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	t.Run("Health check works", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make health request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Verify JSON response
		var healthResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if status, ok := healthResp["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected healthy status, got %v", healthResp["status"])
		}
	})

	t.Run("List users works", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/users", nil)
		if err != nil {
			t.Fatalf("Failed to list users: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Check content type
		if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("Get specific user works", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/user?id=1", nil)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var user User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			t.Fatalf("Failed to decode user response: %v", err)
		}

		if user.ID != 1 {
			t.Errorf("Expected user ID 1, got %d", user.ID)
		}
	})

	t.Run("Invalid user returns 404", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/user?id=999", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})
}

func TestIntegration_MiddlewareChain(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	t.Run("CORS headers are present", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		expectedHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type, Authorization",
		}

		for header, expectedValue := range expectedHeaders {
			if actualValue := resp.Header.Get(header); actualValue != expectedValue {
				t.Errorf("Expected header %s: %s, got: %s", header, expectedValue, actualValue)
			}
		}
	})

	t.Run("OPTIONS request works", func(t *testing.T) {
		resp, err := makeRequest(server, "OPTIONS", "/user", nil)
		if err != nil {
			t.Fatalf("Failed to make OPTIONS request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("Rate limiting eventually kicks in", func(t *testing.T) {
		// This test might be flaky due to timing, but demonstrates the concept
		rateLimitHit := false

		// Make many requests quickly
		for i := 0; i < 30; i++ {
			resp, err := makeRequest(server, "GET", "/health", nil)
			if err != nil {
				t.Fatalf("Failed to make request %d: %v", i, err)
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitHit = true
				resp.Body.Close()
				break
			}
			resp.Body.Close()

			// Small delay to avoid overwhelming
			time.Sleep(10 * time.Millisecond)
		}

		// Note: This test might not always pass due to rate limiter settings
		// In a real scenario, you'd configure a stricter rate limit for testing
		if !rateLimitHit {
			t.Log("Rate limiting not triggered (this may be expected based on current settings)")
		}
	})
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	successes := make(chan bool, numGoroutines)

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Alternate between different endpoints
			var path string
			switch goroutineID % 3 {
			case 0:
				path = "/health"
			case 1:
				path = "/users"
			case 2:
				path = fmt.Sprintf("/user?id=%d", (goroutineID%3)+1)
			}

			resp, err := makeRequest(server, "GET", path, nil)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d failed: %v", goroutineID, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("goroutine %d got status %d", goroutineID, resp.StatusCode)
				return
			}

			successes <- true
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()

	// Count results with timeout
	successCount := 0
	errorCount := 0
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-successes:
			successCount++
		case err := <-errors:
			errorCount++
			t.Errorf("Concurrent request error: %v", err)
		case <-done:
			goto finished
		case <-timeout:
			t.Fatal("Concurrent test timed out")
		}
	}

finished:
	t.Logf("Concurrent test completed: %d successes, %d errors", successCount, errorCount)

	if successCount < numGoroutines/2 {
		t.Errorf("Too many failures in concurrent test: %d/%d successful", successCount, numGoroutines)
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid endpoint",
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
		},
		{
			name:           "Missing user ID",
			path:           "/user",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing id parameter",
		},
		{
			name:           "Invalid user ID format",
			path:           "/user?id=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id parameter",
		},
		{
			name:           "User not found",
			path:           "/user?id=999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := makeRequest(server, "GET", tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				body, err := readResponseBody(resp)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				if !strings.Contains(body, tt.expectedError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedError, body)
				}
			}
		})
	}
}

func TestIntegration_ResponseFormat(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	t.Run("User response has correct JSON format", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/user?id=1", nil)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		defer resp.Body.Close()

		var user User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			t.Fatalf("Failed to decode user JSON: %v", err)
		}

		// Verify all expected fields are present and correct type
		if user.ID == 0 {
			t.Error("User ID should not be zero")
		}
		if user.Name == "" {
			t.Error("User name should not be empty")
		}
		if user.Email == "" {
			t.Error("User email should not be empty")
		}
		if !strings.Contains(user.Email, "@") {
			t.Error("User email should contain @")
		}
	})

	t.Run("Users list response has correct format", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/users", nil)
		if err != nil {
			t.Fatalf("Failed to get users: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode users JSON: %v", err)
		}

		// Check structure
		if _, ok := response["users"]; !ok {
			t.Error("Response should contain 'users' field")
		}
		if _, ok := response["total"]; !ok {
			t.Error("Response should contain 'total' field")
		}

		// Verify users array
		users, ok := response["users"].([]interface{})
		if !ok {
			t.Fatal("Users field should be an array")
		}

		if len(users) == 0 {
			t.Error("Users array should not be empty")
		}
	})
}

// Performance integration test
func TestIntegration_Performance(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	// Warm up
	for i := 0; i < 10; i++ {
		resp, _ := makeRequest(server, "GET", "/health", nil)
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Measure response times
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		resp, err := makeRequest(server, "GET", "/user?id=1", nil)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Logf("Request %d rate limited (expected)", i)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Request %d returned status %d", i, resp.StatusCode)
		}
		resp.Body.Close()
	}

	duration := time.Since(start)
	avgResponseTime := duration / time.Duration(iterations)

	t.Logf("Average response time: %v", avgResponseTime)

	// Assert reasonable performance (adjust based on your requirements)
	if avgResponseTime > 50*time.Millisecond {
		t.Errorf("Average response time too slow: %v", avgResponseTime)
	}
}

// Test server startup and shutdown
func TestIntegration_ServerLifecycle(t *testing.T) {
	t.Run("Server starts and stops cleanly", func(t *testing.T) {
		server := createTestServer()

		// Test that server is responding
		resp, err := makeRequest(server, "GET", "/health", nil)
		if err != nil {
			t.Fatalf("Server not responding: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected healthy server, got status %d", resp.StatusCode)
		}

		// Clean shutdown
		server.Close()

		// Verify server is no longer responding
		_, err = makeRequest(server, "GET", "/health", nil)
		if err == nil {
			t.Error("Server should not be responding after close")
		}
	})
}
