package config

import (
	"fmt"
	"time"
)

// DefaultConfig returns a complete configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Core settings
		Environment: "development",
		ServiceName: "audit-reports-service",
		LogLevel:    "info",
		Version:     "1.0.0",

		// Component configurations with defaults
		Adapters:      DefaultAdapterConfig(),
		HTTP:          DefaultHTTPConfig(),
		Lambda:        DefaultLambdaConfig(),
		Handler:       DefaultHandlerConfig(),
		Retry:         DefaultRetryConfig(),
		Storage:       DefaultStorageConfig(),
		Observability: DefaultObservabilityConfig(),
		RabbitMQ:      DefaultRabbitMQConfig(),
	}
}

// DefaultAdapterConfig returns default adapter selection
func DefaultAdapterConfig() AdapterConfig {
	return AdapterConfig{
		Handler: "http",
		Storage: "filesystem",
		Logger:  "stdout",
		Metrics: "stdout",
	}
}

// DefaultHTTPConfig returns sensible defaults for HTTP configuration
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:    120 * time.Second,
		MaxRetries: 3,
		UserAgent:  "audit-reports-downloader/1.0",
		Addr:       ":8080",
	}
}

// DefaultLambdaConfig returns sensible defaults for Lambda configuration
func DefaultLambdaConfig() LambdaConfig {
	return LambdaConfig{
		Timeout:                   180 * time.Second,
		EnablePartialBatchFailure: true,
	}
}

// DefaultHandlerConfig returns sensible defaults for handler configuration
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		Timeout:        30 * time.Second,
		MaxRequestSize: 10 * 1024 * 1024, // 10MB
		EnableHealth:   true,
		EnableMetrics:  true,
		EnableTracing:  true,
	}
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// DefaultStorageConfig returns sensible defaults for storage configuration
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		BucketOrPath:  "/tmp/storage",
		EnableMetrics: true,
		MaxRetries:    3,
		Timeout:       30 * time.Second,
		S3:            DefaultS3Config(),
	}
}

// DefaultS3Config returns sensible defaults for S3 configuration
func DefaultS3Config() S3Config {
	return S3Config{
		Region: "us-east-2",
	}
}

// DefaultObservabilityConfig returns sensible defaults for observability configuration
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		CloudWatchRegion:    "us-east-2",
		CloudWatchLogGroup:  "",
		CloudWatchNamespace: "",
	}
}

// DefaultRabbitMQConfig returns sensible defaults for RabbitMQ configuration
func DefaultRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		URL:           "amqp://guest:guest@localhost:5672/",
		Queue:         "default-queue",
		PrefetchCount: 10,
		Timeout:       30 * time.Second,
	}
}

// applyDefaults applies environment-specific defaults
func applyDefaults(cfg *Config) {
	// Set adapter defaults based on environment
	if cfg.IsLocal() {
		if cfg.Adapters.Handler == "" {
			cfg.Adapters.Handler = "http"
		}
		if cfg.Adapters.Storage == "" {
			cfg.Adapters.Storage = "filesystem"
		}
		if cfg.Adapters.Logger == "" {
			cfg.Adapters.Logger = "stdout"
		}
		if cfg.Adapters.Metrics == "" {
			cfg.Adapters.Metrics = "stdout"
		}
		if cfg.Storage.BucketOrPath == "" {
			cfg.Storage.BucketOrPath = "/tmp/storage"
		}
	} else if cfg.IsProduction() {
		if cfg.Adapters.Handler == "" {
			cfg.Adapters.Handler = "lambda"
		}
		if cfg.Adapters.Storage == "" {
			cfg.Adapters.Storage = "s3"
		}
		if cfg.Adapters.Logger == "" {
			cfg.Adapters.Logger = "cloudwatch"
		}
		if cfg.Adapters.Metrics == "" {
			cfg.Adapters.Metrics = "cloudwatch"
		}
		// More conservative settings for production
		if cfg.Handler.Timeout < 60*time.Second {
			cfg.Handler.Timeout = 60 * time.Second
		}
		if cfg.Retry.MaxAttempts < 5 {
			cfg.Retry.MaxAttempts = 5
		}
		cfg.Handler.EnableMetrics = true
		cfg.Handler.EnableTracing = true
	}

	// Set bucket/path default if still empty
	if cfg.Storage.BucketOrPath == "" {
		if cfg.Adapters.Storage == "s3" {
			cfg.Storage.BucketOrPath = fmt.Sprintf("%s-storage", cfg.ServiceName)
		} else {
			cfg.Storage.BucketOrPath = "/tmp/storage"
		}
	}
}
