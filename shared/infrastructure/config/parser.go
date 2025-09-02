package config

// parse reads configuration from environment variables
func parse() (*Config, error) {
	cfg := &Config{
		// Core
		Environment: getEnv("ENVIRONMENT", "local"),
		ServiceName: getEnv("SERVICE_NAME", "my-worker"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Version:     getEnv("SERVICE_VERSION", "1.0.0"),

		// Adapter selection
		Adapters: AdapterConfig{
			Runtime:  getEnv("ADAPTER_RUNTIME", ""),
			Storage:  getEnv("ADAPTER_STORAGE", ""),
			Database: getEnv("ADAPTER_DATABASE", ""),
			Logger:   getEnv("ADAPTER_LOGGER", ""),
			Metrics:  getEnv("ADAPTER_METRICS", ""),
		},

		// Database Configuration
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getInt("DB_PORT", 5432),
			Database: getEnv("DB_NAME", "myapp"),
			Username: getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),

			// Connection pool
			MaxOpenConns: getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getInt("DB_MAX_IDLE_CONNS", 5),
		},

		// HTTP Configuration
		HTTP: HTTPConfig{
			Timeout:    getDuration("HTTP_TIMEOUT", "120s"),
			MaxRetries: getInt("HTTP_MAX_RETRIES", 3),
			UserAgent:  getEnv("HTTP_USER_AGENT", "audit-reports-downloader/1.0"),
			Addr:       getEnv("HTTP_ADDR", ":8080"),
		},

		// Lambda Configuration
		Lambda: LambdaConfig{
			Timeout:                   getDuration("LAMBDA_TIMEOUT", "180s"),
			EnablePartialBatchFailure: getBool("LAMBDA_PARTIAL_BATCH_FAILURE", true),
		},

		// Storage Configuration
		Storage: StorageConfig{
			BucketOrPath: getEnv("STORAGE_BUCKET_OR_PATH", ""),
			MaxRetries:   getInt("STORAGE_MAX_RETRIES", 3),
			Timeout:      getDuration("STORAGE_TIMEOUT", "30s"),
			S3: S3Config{
				Region:          getEnv("AWS_REGION", "us-east-2"),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
				Endpoint:        getEnv("S3_ENDPOINT", ""),
			},
		},

		// Observability Configuration
		Observability: ObservabilityConfig{
			CloudWatchRegion:    getEnv("CLOUDWATCH_REGION", getEnv("AWS_REGION", "us-east-2")),
			CloudWatchLogGroup:  getEnv("CLOUDWATCH_LOG_GROUP", ""),
			CloudWatchNamespace: getEnv("CLOUDWATCH_NAMESPACE", ""),
		},

		Queue: QueueConfig{
			Queues: QueueNames{
				Downloader:   getEnv("QUEUE_DOWNLOADER", "downloader"),
				Processor:    getEnv("QUEUE_PROCESSOR", "processor"),
				Extractor:    getEnv("QUEUE_EXTRACTOR", "extractor"),
				DeadLetter:   getEnv("QUEUE_DEAD_LETTER", "dlq"),
				Orchestrator: getEnv("QUEUE_ORCHESTRATOR", "orchestrator"),
			},

			RuntimeQueueName: getEnv("QUEUE_RUNTIME_NAME", ""),

			RabbitMQ: RabbitMQConfig{
				URL:           getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
				PrefetchCount: getInt("RABBITMQ_PREFETCH_COUNT", 10),
				Timeout:       getDuration("RABBITMQ_TIMEOUT", "30s"),
			},

			SQS: SQSConfig{
				Region: getEnv("SQS_REGION", getEnv("AWS_REGION", "us-east-2")),
			},
		},
	}

	return cfg, nil
}
