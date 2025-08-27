package config

import (
	"fmt"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Core settings
	Environment string
	ServiceName string
	LogLevel    string

	// Component configurations
	HTTP    HTTPConfig
	Lambda  LambdaConfig
	Handler HandlerConfig
	Retry   RetryConfig
	Storage StorageConfig
}

// HTTPConfig holds HTTP client configuration
type HTTPConfig struct {
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
	Addr       string // Server address for HTTP mode
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	// S3-specific configuration
	S3 S3Config

	EnableMetrics bool
	MaxRetries    int
	Timeout       time.Duration
	Provider      string // S3, minIO, etc..
}

// S3Config holds S3-specific configuration
type S3Config struct {
	// AWS region (e.g., "us-east-1")
	Region string

	// S3 bucket name for storing objects
	Bucket string

	// AWS credentials (optional if using IAM roles)
	AccessKeyID     string
	SecretAccessKey string
}

// LambdaConfig holds Lambda-specific configuration
type LambdaConfig struct {
	Timeout                   time.Duration
	EnablePartialBatchFailure bool
}

// HandlerConfig holds handler configuration
type HandlerConfig struct {
	Timeout        time.Duration
	MaxRequestSize int64
	EnableHealth   bool
	EnableMetrics  bool
	EnableTracing  bool
	Platform       string // auto-detected if empty
}

// RetryConfig holds retry policy configuration
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	var errors []string
	// Core validations
	if c.ServiceName == "" {
		errors = append(errors, "SERVICE_NAME is required")
	}
	// Range validations
	if c.HTTP.Timeout <= 0 {
		errors = append(errors, "HTTP_TIMEOUT must be positive")
	}
	if c.Handler.Timeout <= 0 {
		errors = append(errors, "HANDLER_TIMEOUT must be positive")
	}
	if c.HTTP.MaxRetries < 0 {
		errors = append(errors, "HTTP_MAX_RETRIES cannot be negative")
	}
	if c.Handler.MaxRequestSize <= 0 {
		errors = append(errors, "HANDLER_MAX_REQUEST_SIZE must be positive")
	}
	if c.Retry.MaxAttempts < 0 {
		errors = append(errors, "RETRY_MAX_ATTEMPTS cannot be negative")
	}
	if c.Retry.BackoffMultiplier < 1.0 {
		errors = append(errors, "RETRY_BACKOFF_MULTIPLIER must be >= 1.0")
	}
	if err := c.Storage.Validate(); err != nil {
		errors = append(errors, fmt.Sprintf("storage config: %v", err))
	}
	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates storage configuration
func (s *StorageConfig) Validate() error {
	if s.MaxRetries < 0 {
		return fmt.Errorf("STORAGE_MAX_RETRIES cannot be negative")
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("STORAGE_TIMEOUT must be positive")
	}

	switch s.GetProvider() {
	case "s3":
		return s.S3.Validate()
	default:
		return fmt.Errorf("unsupported storage provider")
	}
}

func (s *S3Config) Validate() error {
	if s.Bucket == "" {
		return fmt.Errorf("S3_BUCKET is required")
	}

	if s.Region == "" {
		return fmt.Errorf("AWS_REGION is required")
	}

	return nil
}

// applyDefaults applies environment-specific defaults
func (c *Config) applyDefaults() {
	// Apply environment-specific defaults
	if c.IsProduction() {
		// More conservative settings for production
		if c.Handler.Timeout < 60*time.Second {
			c.Handler.Timeout = 60 * time.Second
		}
		if c.Retry.MaxAttempts < 5 {
			c.Retry.MaxAttempts = 5
		}
		// Enable all observability features in production
		c.Handler.EnableMetrics = true
		c.Handler.EnableTracing = true
	}

	if c.IsLocal() {
		// More lenient settings for local development
		c.Handler.EnableTracing = false // No need for tracing locally
	}
}

// Environment detection methods

// IsLocal returns true if running in local/development environment
func (c *Config) IsLocal() bool {
	env := strings.ToLower(c.Environment)
	return env == "local" || env == "development" || env == "dev"
}

// IsStaging returns true if running in staging environment
func (c *Config) IsStaging() bool {
	env := strings.ToLower(c.Environment)
	return env == "staging" || env == "stage"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	env := strings.ToLower(c.Environment)
	return env == "production" || env == "prod"
}

// IsTest returns true if running in test environment
func (c *Config) IsTest() bool {
	env := strings.ToLower(c.Environment)
	return env == "test" || env == "testing"
}

// IsStorageEnabled returns true if any storage provider is configured
func (c *Config) IsStorageEnabled() bool {
	return c.Storage.Provider != ""
}

func (s *StorageConfig) GetProvider() string {
	switch strings.ToLower(s.Provider) {
	case "s3":
		return "s3"
	default:
		return "unknown"
	}
}

// GetStorageProvider returns the configured storage provider
func (c *Config) GetStorageProvider() string {
	return c.Storage.GetProvider()
}
