package config

import (
	"time"
)

// Config holds all application configuration
type Config struct {
	// Core settings
	Environment string
	ServiceName string
	LogLevel    string
	Version     string

	// Adapter selection
	Adapters AdapterConfig

	// Component configurations
	HTTP          HTTPConfig
	Lambda        LambdaConfig
	Runtime       RuntimeConfig
	Retry         RetryConfig
	Storage       StorageConfig
	Database      DatabaseConfig
	Observability ObservabilityConfig
	RabbitMQ      RabbitMQConfig
}

// AdapterConfig specifies which implementations to use
type AdapterConfig struct {
	Runtime  string // "lambda", "http", "rabbitmq"
	Storage  string // "s3", "filesystem"
	Database string // "postgres"
	Logger   string // "cloudwatch", "stdout"
	Metrics  string // "cloudwatch", "stdout"
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	// Connection settings
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	MaxOpenConns int
	MaxIdleConns int
	SSLMode      string
}

// HTTPConfig holds HTTP configuration
type HTTPConfig struct {
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
	Addr       string // Server address for HTTP mode
}

// LambdaConfig holds Lambda-specific configuration
type LambdaConfig struct {
	Timeout                   time.Duration
	EnablePartialBatchFailure bool
}

// RuntimeConfig holds handler configuration
type RuntimeConfig struct {
	Timeout        time.Duration
	MaxRequestSize int64
	EnableHealth   bool
	EnableMetrics  bool
	EnableTracing  bool
}

// RetryConfig holds retry policy configuration
type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	// Common fields for all storage types
	BucketOrPath  string
	EnableMetrics bool
	MaxRetries    int
	Timeout       time.Duration

	// S3-specific configuration
	S3 S3Config
}

// S3Config holds S3-specific configuration
type S3Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // For MinIO or S3-compatible services
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	CloudWatchRegion    string
	CloudWatchLogGroup  string
	CloudWatchNamespace string
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	URL           string
	Queue         string
	PrefetchCount int
	Timeout       time.Duration
}
