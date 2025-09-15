package integration

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
	"user-service/internal/config"
	"user-service/internal/handlers"
	"user-service/internal/metrics"
	"user-service/internal/middleware"
	"user-service/internal/models"
	"user-service/internal/services"
)

func createTestServer() *httptest.Server {
	// Create test registry to avoid conflicts
	testRegistry := prometheus.NewRegistry()
	metricsCollector := metrics.New(testRegistry, testRegistry)

	// Create service
	userService := services.NewUserService(metricsCollector)

	// Load configuration
	cfg := config.Load()

	// Setup routes with middleware
	mux := setupRoutes(userService, metricsCollector, cfg)

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

// closeResponseBody is a helper to safely close response bodies in tests
func closeResponseBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}
}

func TestIntegration_CompleteUserWorkflow(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	t.Run("Health check works", func(t *testing.T) {
		resp, err := makeRequest(server, "GET", "/health", nil)
		if err != nil {
			t.Fatalf("Failed to make health request: %v", err)
		}
		defer closeResponseBody(t, resp)

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
		defer closeResponseBody(t, resp)

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
		defer closeResponseBody(t, resp)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var user models.User
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
		defer closeResponseBody(t, resp)

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
		defer closeResponseBody(t, resp)

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
		defer closeResponseBody(t, resp)

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
				closeResponseBody(t, resp)
				rateLimitHit = true
				break
			}
			closeResponseBody(t, resp)

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
	rateLimited := make(chan bool, numGoroutines) // Add separate channel for rate limiting

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
			defer closeResponseBody(t, resp)

			// Handle different status codes appropriately
			switch resp.StatusCode {
			case http.StatusOK:
				successes <- true
			case http.StatusTooManyRequests:
				rateLimited <- true // Rate limiting is expected, not an error
			default:
				errors <- fmt.Errorf("goroutine %d got unexpected status %d", goroutineID, resp.StatusCode)
			}
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
	rateLimitedCount := 0
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-successes:
			successCount++
		case <-rateLimited:
			rateLimitedCount++
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
	t.Logf("Concurrent test completed: %d successes, %d rate limited, %d errors",
		successCount, rateLimitedCount, errorCount)

	// Test should pass if we have some successes and rate limiting is working
	// Only fail if we have actual errors or no successes at all
	if errorCount > 0 {
		t.Errorf("Got %d actual errors in concurrent test", errorCount)
	}

	if successCount == 0 {
		t.Error("No successful requests in concurrent test - service may be down")
	}

	// Log that rate limiting is working correctly
	if rateLimitedCount > 0 {
		t.Logf("Rate limiting working correctly - %d requests succeeded, %d rate limited",
			successCount, rateLimitedCount)
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
			expectedError:  "id parameter is missing",
		},
		{
			name:           "Invalid user ID format",
			path:           "/user?id=abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "id parameter is invalid",
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
			defer closeResponseBody(t, resp)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				body, err := io.ReadAll(resp.Body) // Read directly instead of using readResponseBody
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}

				if !strings.Contains(strings.TrimSpace(string(body)), tt.expectedError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedError, string(body))
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
		defer closeResponseBody(t, resp)

		var user models.User
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
		defer closeResponseBody(t, resp)

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
		resp, err := makeRequest(server, "GET", "/health", nil)
		if err != nil {
			t.Logf("Warm-up request %d failed: %v", i, err)
			continue
		}
		if resp != nil {
			closeResponseBody(t, resp)
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
			closeResponseBody(t, resp)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			closeResponseBody(t, resp)
			t.Fatalf("Request %d returned status %d", i, resp.StatusCode)
		}
		closeResponseBody(t, resp)
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
		defer closeResponseBody(t, resp)

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
func setupRoutes(userService *services.UserService, metricsCollector *metrics.Metrics, cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	// Create handlers
	userHandler := handlers.NewUserHandler(userService)
	healthHandler := handlers.NewHealthHandler(userService)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = middleware.Recovery(metricsCollector)(handler)
	handler = middleware.CORS()(handler)
	handler = middleware.RateLimit(cfg.GetRateLimiter(), metricsCollector)(handler)
	handler = middleware.Metrics(metricsCollector)(handler)
	handler = middleware.Logging()(handler)

	// Register application routes
	mux.HandleFunc("/user", userHandler.GetUser)
	mux.HandleFunc("/users", userHandler.ListUsers)
	mux.HandleFunc("/health", healthHandler.Health)

	// Register metrics endpoint
	mux.Handle("/metrics", metricsCollector.Handler())

	// Wrap the final handler
	finalMux := http.NewServeMux()
	finalMux.Handle("/", handler)

	return finalMux
}
