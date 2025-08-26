package config

import "time"

// DefaultHandlerConfig returns sensible defaults for handler configuration
func DefaultHandlerConfig() HandlerConfig {
	return HandlerConfig{
		Timeout:        30 * time.Second,
		MaxRequestSize: 10 * 1024 * 1024, // 10MB
		EnableHealth:   true,
		EnableMetrics:  true,
		EnableTracing:  true,
		Platform:       "", // Auto-detect
	}
}

// DefaultHTTPConfig returns sensible defaults for HTTP client configuration
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:    120 * time.Second,
		MaxRetries: 3,
		UserAgent:  "audit-reports-downloader/1.0",
		Addr:       ":8080",
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

// DefaultAWSConfig returns sensible defaults for AWS configuration
func DefaultAWSConfig() AWSConfig {
	return AWSConfig{
		Region:             "us-east-1",
		LocalStackEndpoint: "",
	}
}

// DefaultStorageConfig returns sensible defaults for storage configuration
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		S3Bucket: "", // Must be set by environment or override
	}
}

// DefaultQueueConfig returns sensible defaults for queue configuration
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		DownloaderQueue: "", // Must be set by environment or override
		DLQName:         "", // Must be set by environment or override
		ProcessorQueue:  "", // Must be set by environment or override
	}
}

// DefaultLambdaConfig returns sensible defaults for Lambda configuration
func DefaultLambdaConfig() LambdaConfig {
	return LambdaConfig{
		Timeout:                   180 * time.Second,
		MemorySize:                512,
		MaxConcurrency:            10,
		ProcessingTimeout:         30 * time.Second,
		EnablePartialBatchFailure: true,
		AutoBase64Decode:          true,
	}
}

// DefaultConfig returns a complete configuration with sensible defaults
// This is useful for testing or when you want to start with defaults and override specific parts
func DefaultConfig() *Config {
	return &Config{
		// Core settings
		Environment: "development",
		ServiceName: "audit-reports-service",
		LogLevel:    "info",

		// Component configurations with defaults
		AWS:     DefaultAWSConfig(),
		HTTP:    DefaultHTTPConfig(),
		Storage: DefaultStorageConfig(),
		Queue:   DefaultQueueConfig(),
		Lambda:  DefaultLambdaConfig(),
		Handler: DefaultHandlerConfig(),
		Retry:   DefaultRetryConfig(),
	}
}
