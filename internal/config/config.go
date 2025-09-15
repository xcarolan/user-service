package config

import (
	"os"
	"strconv"

	"golang.org/x/time/rate"
)

type Config struct {
	Port        string
	LogLevel    string
	DatabaseURL string
	RateLimit   struct {
		RequestsPerSecond float64
		BurstSize         int
	}
}

func Load() *Config {
	cfg := &Config{
		Port:        getEnv("PORT", ":8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@postgres:5432/user_service?sslmode=disable"),
	}

	// Rate limiting configuration
	cfg.RateLimit.RequestsPerSecond = getEnvFloat("RATE_LIMIT_RPS", 10.0)
	cfg.RateLimit.BurstSize = getEnvInt("RATE_LIMIT_BURST", 20)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func (c *Config) GetRateLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(c.RateLimit.RequestsPerSecond), c.RateLimit.BurstSize)
}
