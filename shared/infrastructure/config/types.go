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
	Storage       StorageConfig
	Database      DatabaseConfig
	Observability ObservabilityConfig
	Queue         QueueConfig
}

// AdapterConfig specifies which implementations to use
type AdapterConfig struct {
	Runtime  string // "lambda", "http", "rabbitmq"
	Storage  string // "s3", "filesystem"
	Database string // "postgres"
	Logger   string // "cloudwatch", "stdout"
	Metrics  string // "cloudwatch", "stdout"
	Queue    string // "rabbitmq", "sqs" - for publishing
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

type StorageConfig struct {
	// Common fields for all storage types
	BucketOrPath string
	MaxRetries   int
	Timeout      time.Duration

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

// QueueConfig holds minimal queue configuration
type QueueConfig struct {
	Queues           QueueNames
	RuntimeQueueName string

	// Connection settings based on adapter
	RabbitMQ RabbitMQConfig
	SQS      SQSConfig
}

// QueueNames defines all queue names in the system
type QueueNames struct {
	Downloader   string // Queue for download tasks
	Processor    string // Queue for processing tasks
	Extractor    string // Queue for extraction tasks
	DeadLetter   string // Dead letter queue for failed messages
	Orchestrator string // Queue for orchestration tasks
}

// RabbitMQConfig - minimal config
type RabbitMQConfig struct {
	URL           string // Connection URL
	Timeout       time.Duration
	PrefetchCount int
}

// SQSConfig - minimal config
type SQSConfig struct {
	Region string // AWS Region
}
