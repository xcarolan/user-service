package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg := Load()
	if cfg.Port != ":8080" {
		t.Errorf("Expected Port to be :8080, got %s", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be info, got %s", cfg.LogLevel)
	}
	if cfg.RateLimit.RequestsPerSecond != 10.0 {
		t.Errorf("Expected RateLimit.RequestsPerSecond to be 10.0, got %f", cfg.RateLimit.RequestsPerSecond)
	}
	if cfg.RateLimit.BurstSize != 20 {
		t.Errorf("Expected RateLimit.BurstSize to be 20, got %d", cfg.RateLimit.BurstSize)
	}

	// Test with environment variables
	if err := os.Setenv("PORT", ":9090"); err != nil {
		t.Fatalf("Failed to set PORT: %v", err)
	}
	if err := os.Setenv("LOG_LEVEL", "debug"); err != nil {
		t.Fatalf("Failed to set LOG_LEVEL: %v", err)
	}
	if err := os.Setenv("RATE_LIMIT_RPS", "100.0"); err != nil {
		t.Fatalf("Failed to set RATE_LIMIT_RPS: %v", err)
	}
	if err := os.Setenv("RATE_LIMIT_BURST", "200"); err != nil {
		t.Fatalf("Failed to set RATE_LIMIT_BURST: %v", err)
	}

	cfg = Load()
	if cfg.Port != ":9090" {
		t.Errorf("Expected Port to be :9090, got %s", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be debug, got %s", cfg.LogLevel)
	}
	if cfg.RateLimit.RequestsPerSecond != 100.0 {
		t.Errorf("Expected RateLimit.RequestsPerSecond to be 100.0, got %f", cfg.RateLimit.RequestsPerSecond)
	}
	if cfg.RateLimit.BurstSize != 200 {
		t.Errorf("Expected RateLimit.BurstSize to be 200, got %d", cfg.RateLimit.BurstSize)
	}

	// Clean up environment variables
	if err := os.Unsetenv("PORT"); err != nil {
		t.Logf("Warning: failed to unset PORT: %v", err)
	}
	if err := os.Unsetenv("LOG_LEVEL"); err != nil {
		t.Logf("Warning: failed to unset LOG_LEVEL: %v", err)
	}
	if err := os.Unsetenv("RATE_LIMIT_RPS"); err != nil {
		t.Logf("Warning: failed to unset RATE_LIMIT_RPS: %v", err)
	}
	if err := os.Unsetenv("RATE_LIMIT_BURST"); err != nil {
		t.Logf("Warning: failed to unset RATE_LIMIT_BURST: %v", err)
	}
}
