package config

import (
	"fmt"
	"strings"
)

// Validate validates the entire configuration
func (c *Config) Validate() error {
	var errors []string

	// Core validations
	if c.ServiceName == "" {
		errors = append(errors, "SERVICE_NAME is required")
	}

	// Validate adapters
	if err := c.Adapters.Validate(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate handler config
	if err := c.Runtime.Validate(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate component configs based on selected adapters
	switch c.Adapters.Runtime {
	case "http":
		if err := c.HTTP.Validate(); err != nil {
			errors = append(errors, err.Error())
		}
	case "lambda":
		if err := c.Lambda.Validate(); err != nil {
			errors = append(errors, err.Error())
		}
	case "rabbitmq":
		if err := c.RabbitMQ.Validate(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate storage
	if err := c.Storage.Validate(c.Adapters); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate observability if using CloudWatch
	if c.Adapters.Logger == "cloudwatch" || c.Adapters.Metrics == "cloudwatch" {
		if err := c.Observability.Validate(c.Adapters); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Validate retry config
	if err := c.Retry.Validate(); err != nil {
		errors = append(errors, err.Error())
	}

	// Validate database if configured
	if c.Adapters.Database != "" {
		validDatabases := map[string]bool{"postgres": true}

		if !validDatabases[c.Adapters.Database] {
			errors = append(errors, fmt.Sprintf("invalid database adapter: %s", c.Adapters.Database))
		}

		if err := c.Database.Validate(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Validate validates adapter configuration
func (a *AdapterConfig) Validate() error {
	validHandlers := map[string]bool{"lambda": true, "http": true, "rabbitmq": true}
	if !validHandlers[a.Runtime] {
		return fmt.Errorf("invalid handler adapter: %s (must be lambda, http, or rabbitmq)", a.Runtime)
	}

	validStorage := map[string]bool{"s3": true, "filesystem": true}
	if !validStorage[a.Storage] {
		return fmt.Errorf("invalid storage adapter: %s (must be s3 or filesystem)", a.Storage)
	}

	validLogger := map[string]bool{"cloudwatch": true, "stdout": true}
	if !validLogger[a.Logger] {
		return fmt.Errorf("invalid logger adapter: %s (must be cloudwatch or stdout)", a.Logger)
	}

	validMetrics := map[string]bool{"cloudwatch": true, "stdout": true}
	if !validMetrics[a.Metrics] {
		return fmt.Errorf("invalid metrics adapter: %s (must be cloudwatch or stdout)", a.Metrics)
	}

	return nil
}

// Validate validates HTTP configuration
func (h *HTTPConfig) Validate() error {
	if h.Timeout <= 0 {
		return fmt.Errorf("HTTP_TIMEOUT must be positive")
	}
	if h.MaxRetries < 0 {
		return fmt.Errorf("HTTP_MAX_RETRIES cannot be negative")
	}
	if h.Addr == "" {
		return fmt.Errorf("HTTP_ADDR is required for HTTP adapter")
	}
	return nil
}

// Validate validates Lambda configuration
func (l *LambdaConfig) Validate() error {
	if l.Timeout <= 0 {
		return fmt.Errorf("LAMBDA_TIMEOUT must be positive")
	}
	return nil
}

// Validate validates Handler configuration
func (h *RuntimeConfig) Validate() error {
	if h.Timeout <= 0 {
		return fmt.Errorf("HANDLER_TIMEOUT must be positive")
	}
	if h.MaxRequestSize <= 0 {
		return fmt.Errorf("HANDLER_MAX_REQUEST_SIZE must be positive")
	}
	return nil
}

// Validate validates RabbitMQ configuration
func (r *RabbitMQConfig) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("RABBITMQ_URL is required for RabbitMQ adapter")
	}
	if r.Queue == "" {
		return fmt.Errorf("RABBITMQ_QUEUE is required for RabbitMQ adapter")
	}
	if r.PrefetchCount < 0 {
		return fmt.Errorf("RABBITMQ_PREFETCH_COUNT cannot be negative")
	}
	if r.Timeout <= 0 {
		return fmt.Errorf("RABBITMQ_TIMEOUT must be positive")
	}
	return nil
}

// Validate validates Storage configuration
func (s *StorageConfig) Validate(adapters AdapterConfig) error {
	if s.MaxRetries < 0 {
		return fmt.Errorf("STORAGE_MAX_RETRIES cannot be negative")
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("STORAGE_TIMEOUT must be positive")
	}

	// Validate based on selected storage adapter
	switch adapters.Storage {
	case "s3":
		if s.BucketOrPath == "" {
			return fmt.Errorf("STORAGE_BUCKET_OR_PATH (bucket) is required for S3 storage")
		}
		if s.S3.Region == "" {
			return fmt.Errorf("AWS_REGION is required for S3 storage")
		}
	case "filesystem":
		if s.BucketOrPath == "" {
			return fmt.Errorf("STORAGE_BUCKET_OR_PATH (path) is required for filesystem storage")
		}
	}

	return nil
}

// Validate validates Observability configuration
func (o *ObservabilityConfig) Validate(adapters AdapterConfig) error {
	if adapters.Logger == "cloudwatch" || adapters.Metrics == "cloudwatch" {
		if o.CloudWatchRegion == "" {
			return fmt.Errorf("CLOUDWATCH_REGION is required for CloudWatch")
		}
	}

	if adapters.Logger == "cloudwatch" {
		if o.CloudWatchLogGroup == "" {
			return fmt.Errorf("CLOUDWATCH_LOG_GROUP is required for CloudWatch logging")
		}
	}

	if adapters.Metrics == "cloudwatch" {
		if o.CloudWatchNamespace == "" {
			return fmt.Errorf("CLOUDWATCH_NAMESPACE is required for CloudWatch metrics")
		}
	}

	return nil
}

// Validate validates Retry configuration
func (r *RetryConfig) Validate() error {
	if r.MaxAttempts < 0 {
		return fmt.Errorf("RETRY_MAX_ATTEMPTS cannot be negative")
	}
	if r.InitialBackoff <= 0 {
		return fmt.Errorf("RETRY_INITIAL_BACKOFF must be positive")
	}
	if r.MaxBackoff <= 0 {
		return fmt.Errorf("RETRY_MAX_BACKOFF must be positive")
	}
	if r.BackoffMultiplier < 1.0 {
		return fmt.Errorf("RETRY_BACKOFF_MULTIPLIER must be >= 1.0")
	}
	return nil
}

// Validate validates Database configuration
func (d *DatabaseConfig) Validate() error {
	var errors []string

	// Required connection settings
	if d.Host == "" {
		errors = append(errors, "DB_HOST is required")
	}

	if d.Port <= 0 || d.Port > 65535 {
		errors = append(errors, "DB_PORT must be between 1 and 65535")
	}

	if d.Database == "" {
		errors = append(errors, "DB_NAME is required")
	}

	if d.Username == "" {
		errors = append(errors, "DB_USER is required")
	}

	// Password can be empty for some local setups, so we don't validate it

	// Optional connection pool validation (only if set)
	if d.MaxOpenConns < 0 {
		errors = append(errors, "DB_MAX_OPEN_CONNS cannot be negative")
	}

	if d.MaxIdleConns < 0 {
		errors = append(errors, "DB_MAX_IDLE_CONNS cannot be negative")
	}

	if d.MaxOpenConns > 0 && d.MaxIdleConns > d.MaxOpenConns {
		errors = append(errors, "DB_MAX_IDLE_CONNS cannot be greater than DB_MAX_OPEN_CONNS")
	}

	if len(errors) > 0 {
		return fmt.Errorf("database configuration errors: %s", strings.Join(errors, "; "))
	}

	return nil
}
