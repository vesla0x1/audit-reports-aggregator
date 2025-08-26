package handler

import (
	"os"
	"strconv"
	"time"
)

// Config holds handler configuration.
// This centralizes all configuration options for the handler,
// making it easy to customize behavior for different environments.
type Config struct {
	// Timeout for request processing
	Timeout time.Duration

	// MaxRequestSize in bytes (default: 10MB)
	MaxRequestSize int64

	// EnableHealthCheck enables health endpoint
	EnableHealthCheck bool

	// EnableMetrics enables metrics collection
	EnableMetrics bool

	// EnableTracing enables distributed tracing
	EnableTracing bool

	// Environment (development, staging, production)
	Environment string

	// Platform identifier (http, KEDA, lambda)
	Platform string

	// RetryConfig for retry middleware
	RetryConfig *RetryConfig

	// RateLimitConfig for rate limiting
	RateLimitConfig *RateLimitConfig
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration

	// BackoffMultiplier for exponential backoff
	BackoffMultiplier float64
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	// RequestsPerSecond is the rate limit
	RequestsPerSecond int

	// BurstSize is the token bucket size
	BurstSize int
}

// DefaultConfig returns default handler configuration.
// These defaults are suitable for most use cases but can be overridden.
func DefaultConfig() *Config {
	return &Config{
		Timeout:           30 * time.Second,
		MaxRequestSize:    10 * 1024 * 1024, // 10MB
		EnableHealthCheck: true,
		EnableMetrics:     true,
		EnableTracing:     true,
		Environment:       "development",
		Platform:          "http",
		RetryConfig:       DefaultRetryConfig(),
		RateLimitConfig:   nil, // Disabled by default
	}
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// ConfigFromEnv creates configuration from environment variables.
// This allows configuration through environment variables without code changes.
func ConfigFromEnv() *Config {
	config := DefaultConfig()

	// Timeout
	if val := os.Getenv("HANDLER_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Timeout = duration
		}
	}

	// Max request size
	if val := os.Getenv("HANDLER_MAX_REQUEST_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.MaxRequestSize = size
		}
	}

	// Features
	if val := os.Getenv("HANDLER_ENABLE_HEALTH"); val != "" {
		config.EnableHealthCheck = val == "true" || val == "1"
	}

	if val := os.Getenv("HANDLER_ENABLE_METRICS"); val != "" {
		config.EnableMetrics = val == "true" || val == "1"
	}

	if val := os.Getenv("HANDLER_ENABLE_TRACING"); val != "" {
		config.EnableTracing = val == "true" || val == "1"
	}

	// Environment
	if val := os.Getenv("ENVIRONMENT"); val != "" {
		config.Environment = val
	} else if val := os.Getenv("ENV"); val != "" {
		config.Environment = val
	}

	// Platform
	if val := os.Getenv("HANDLER_PLATFORM"); val != "" {
		config.Platform = val
	}

	// Retry configuration
	if val := os.Getenv("HANDLER_MAX_RETRIES"); val != "" {
		if retries, err := strconv.Atoi(val); err == nil {
			config.RetryConfig.MaxRetries = retries
		}
	}

	return config
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	if c.MaxRequestSize <= 0 {
		c.MaxRequestSize = 10 * 1024 * 1024
	}

	if c.Environment == "" {
		c.Environment = "development"
	}

	if c.Platform == "" {
		c.Platform = "http"
	}

	return nil
}
