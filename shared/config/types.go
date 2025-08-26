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
	AWS     AWSConfig
	HTTP    HTTPConfig
	Storage StorageConfig
	Queue   QueueConfig
	Lambda  LambdaConfig
	Handler HandlerConfig
	Retry   RetryConfig
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region             string
	LocalStackEndpoint string // Only for local development
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
	S3Bucket string
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	DownloaderQueue string
	DLQName         string
	ProcessorQueue  string
}

// LambdaConfig holds Lambda-specific configuration
type LambdaConfig struct {
	Timeout                   time.Duration
	MemorySize                int
	MaxConcurrency            int
	ProcessingTimeout         time.Duration
	EnablePartialBatchFailure bool
	AutoBase64Decode          bool
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

	// Production-specific validations
	if c.IsProduction() {
		if c.Storage.S3Bucket == "" {
			errors = append(errors, "S3_BUCKET is required in production")
		}
		if c.Queue.DownloaderQueue == "" {
			errors = append(errors, "SQS_QUEUE_NAME is required in production")
		}
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
	if c.Lambda.MemorySize < 128 || c.Lambda.MemorySize > 10240 {
		errors = append(errors, "LAMBDA_MEMORY_SIZE must be between 128 and 10240 MB")
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

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// applyDefaults applies environment-specific defaults
func (c *Config) applyDefaults() {
	env := strings.ToLower(c.Environment)

	// Generate default resource names if not provided
	if c.Storage.S3Bucket == "" {
		c.Storage.S3Bucket = fmt.Sprintf("audit-reports-%s-downloads", env)
	}
	if c.Queue.DownloaderQueue == "" {
		c.Queue.DownloaderQueue = fmt.Sprintf("audit-reports-%s-downloader", env)
	}
	if c.Queue.DLQName == "" {
		c.Queue.DLQName = fmt.Sprintf("audit-reports-%s-downloader-dlq", env)
	}
	if c.Queue.ProcessorQueue == "" {
		c.Queue.ProcessorQueue = fmt.Sprintf("audit-reports-%s-processor", env)
	}

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
