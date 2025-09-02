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
		Storage:       DefaultStorageConfig(),
		Database:      DefaultDatabaseConfig(),
		Observability: DefaultObservabilityConfig(),
		Queue:         DefaultQueueConfig(),
	}
}

// DefaultAdapterConfig returns default adapter selection
func DefaultAdapterConfig() AdapterConfig {
	return AdapterConfig{
		Runtime:  "http",
		Storage:  "filesystem",
		Database: "postgres",
		Logger:   "stdout",
		Metrics:  "stdout",
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

// DefaultStorageConfig returns sensible defaults for storage configuration
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		BucketOrPath: "audit-reports-local",
		MaxRetries:   3,
		Timeout:      30 * time.Second,
		S3:           DefaultS3Config(),
	}
}

// DefaultS3Config returns sensible defaults for S3 configuration
func DefaultS3Config() S3Config {
	return S3Config{
		Region: "us-east-2",
	}
}

// DefaultDatabaseConfig returns sensible defaults for database configuration
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		Database:     "audit_reports_aggregator",
		Username:     "postgres",
		Password:     "postgres",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		SSLMode:      "disable",
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

// DefaultQueueConfig returns sensible defaults for queue configuration
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		Queues: QueueNames{
			Downloader:   "downloader",
			Processor:    "processor",
			Extractor:    "extractor",
			DeadLetter:   "dlq",
			Orchestrator: "orchestrator",
		},
		RabbitMQ: DefaultRabbitMQConfig(),
		SQS:      DefaultSQSConfig(),
	}
}

// DefaultRabbitMQConfig returns sensible defaults for RabbitMQ
func DefaultRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		URL:           "amqp://guest:guest@localhost:5672/",
		Timeout:       30 * time.Second,
		PrefetchCount: 10,
	}
}

// DefaultSQSConfig returns sensible defaults for SQS
func DefaultSQSConfig() SQSConfig {
	return SQSConfig{
		Region: "us-east-2",
	}
}

// applyDefaults applies environment-specific defaults
func applyDefaults(cfg *Config) {
	// Set adapter defaults based on environment
	if cfg.IsLocal() {
		if cfg.Adapters.Runtime == "" {
			cfg.Adapters.Runtime = "rabbitmq"
		}
		if cfg.Adapters.Storage == "" {
			cfg.Adapters.Storage = "filesystem"
		}
		if cfg.Adapters.Database == "" {
			cfg.Adapters.Database = "postgres"
		}
		if cfg.Adapters.Logger == "" {
			cfg.Adapters.Logger = "stdout"
		}
		if cfg.Adapters.Metrics == "" {
			cfg.Adapters.Metrics = "stdout"
		}
		if cfg.Storage.BucketOrPath == "" {
			cfg.Storage.BucketOrPath = "audit-reports-local"
		}
	} else if cfg.IsProduction() {
		if cfg.Adapters.Runtime == "" {
			cfg.Adapters.Runtime = "lambda"
		}
		if cfg.Adapters.Storage == "" {
			cfg.Adapters.Storage = "s3"
		}
		if cfg.Adapters.Database == "" {
			cfg.Adapters.Database = "postgres"
		}
		if cfg.Adapters.Logger == "" {
			cfg.Adapters.Logger = "cloudwatch"
		}
		if cfg.Adapters.Metrics == "" {
			cfg.Adapters.Metrics = "cloudwatch"
		}
		if cfg.Adapters.Queue == "" {
			cfg.Adapters.Queue = "sqs"
		}
	}

	// Set bucket/path default if still empty
	if cfg.Storage.BucketOrPath == "" {
		if cfg.Adapters.Storage == "s3" {
			cfg.Storage.BucketOrPath = fmt.Sprintf("%s-storage", cfg.ServiceName)
		} else {
			cfg.Storage.BucketOrPath = "audit-reports-local"
		}
	}
}
