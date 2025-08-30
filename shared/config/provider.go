package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Provider manages configuration lifecycle and ensures singleton behavior
type Provider struct {
	config *Config
	mu     sync.RWMutex
	loaded bool
}

var (
	instance *Provider
	once     sync.Once
)

// GetProvider returns the singleton configuration provider instance
func GetProvider() *Provider {
	once.Do(func() {
		instance = &Provider{}
	})
	return instance
}

// Load loads configuration from environment variables and .env files
// This should be called once at application startup
func (p *Provider) Load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.loaded {
		return nil // Already loaded
	}

	// Load .env files in order of precedence
	//if err := p.loadEnvFiles(); err != nil {
	//	return fmt.Errorf("failed to load env files: %w", err)
	//}

	// Parse configuration from environment
	cfg, err := p.parseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.config = cfg
	p.loaded = true
	return nil
}

// MustLoad loads configuration and panics on error
// Use this for application initialization where errors are fatal
func (p *Provider) MustLoad() {
	if err := p.Load(); err != nil {
		panic(fmt.Sprintf("failed to load configuration: %v", err))
	}
}

// Get returns the current configuration
// Returns error if configuration hasn't been loaded
func (p *Provider) Get() (*Config, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.loaded || p.config == nil {
		return nil, fmt.Errorf("configuration not loaded; call Load() first")
	}

	return p.config, nil
}

// MustGet returns the configuration or panics if not loaded
// Use this when you're certain configuration has been loaded
func (p *Provider) MustGet() *Config {
	cfg, err := p.Get()
	if err != nil {
		panic(fmt.Sprintf("failed to get configuration: %v", err))
	}
	return cfg
}

// Reload reloads configuration from environment
// Useful for configuration updates without restart (use with caution)
func (p *Provider) Reload() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Parse configuration from current environment
	cfg, err := p.parseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.config = cfg
	return nil
}

// loadEnvFiles loads .env files in order of precedence
func (p *Provider) loadEnvFiles() error {
	// Load base .env file (optional)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return fmt.Errorf("failed to load .env: %w", err)
		}
	}

	// Load environment-specific file (optional)
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env != "" {
		envFile := fmt.Sprintf(".env.%s", env)
		if _, err := os.Stat(envFile); err == nil {
			// Overload (don't override) allows environment-specific values to take precedence
			if err := godotenv.Overload(envFile); err != nil {
				return fmt.Errorf("failed to load %s: %w", envFile, err)
			}
		}
	}

	// Load .env.local for local overrides (highest precedence, optional)
	if _, err := os.Stat(".env.local"); err == nil {
		if err := godotenv.Overload(".env.local"); err != nil {
			return fmt.Errorf("failed to load .env.local: %w", err)
		}
	}

	return nil
}

// parseConfig parses configuration from environment variables
func (p *Provider) parseConfig() (*Config, error) {
	cfg := &Config{
		// Core
		Environment: getEnv("ENVIRONMENT", "local"),
		ServiceName: getEnv("PROJECT_NAME", "my-worker"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Version:     getEnv("SERVICE_VERSION", "1.0.0"),

		// HTTP Client
		HTTP: HTTPConfig{
			Timeout:    getDuration("HTTP_TIMEOUT", "120s"),
			MaxRetries: getInt("HTTP_MAX_RETRIES", 3),
			UserAgent:  getEnv("HTTP_USER_AGENT", "audit-reports-downloader/1.0"),
			Addr:       getEnv("HTTP_ADDR", ":8080"),
		},

		// Lambda
		Lambda: LambdaConfig{
			Timeout:                   getDuration("LAMBDA_TIMEOUT", "180s"),
			EnablePartialBatchFailure: getBool("LAMBDA_PARTIAL_BATCH_FAILURE", true),
		},

		// Handler
		Handler: HandlerConfig{
			Timeout:        getDuration("HANDLER_TIMEOUT", "30s"),
			MaxRequestSize: int64(getInt("HANDLER_MAX_REQUEST_SIZE", 10*1024*1024)),
			EnableHealth:   getBool("HANDLER_ENABLE_HEALTH", true),
			EnableMetrics:  getBool("HANDLER_ENABLE_METRICS", true),
			EnableTracing:  getBool("HANDLER_ENABLE_TRACING", true),
			Platform:       getEnv("HANDLER_PLATFORM", ""),
		},

		// Retry
		Retry: RetryConfig{
			MaxAttempts:       getInt("RETRY_MAX_ATTEMPTS", 3),
			InitialBackoff:    getDuration("RETRY_INITIAL_BACKOFF", "100ms"),
			MaxBackoff:        getDuration("RETRY_MAX_BACKOFF", "10s"),
			BackoffMultiplier: getFloat64("RETRY_BACKOFF_MULTIPLIER", 2.0),
		},

		// Storage
		Storage: StorageConfig{
			EnableMetrics: getBool("STORAGE_ENABLE_METRICS", true),
			MaxRetries:    getInt("STORAGE_MAX_RETRIES", 3),
			Provider:      getEnv("STORAGE_PROVIDER", "s3"),
			Timeout:       getDuration("STORAGE_TIMEOUT", "30s"),
			S3: S3Config{
				Region:          getEnv("AWS_REGION", "us-east-2"),
				Bucket:          getEnv("S3_BUCKET", ""),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", "test"),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", "test"),
			},
		},

		Observability: ObservabilityConfig{
			LogProvider:         getEnv("OBSERVABILITY_LOG_PROVIDER", "console"),
			MetricsProvider:     getEnv("OBSERVABILITY_METRICS_PROVIDER", "noop"),
			CloudWatchRegion:    getEnv("OBSERVABILITY_CLOUDWATCH_REGION", getEnv("AWS_REGION", "us-east-2")),
			CloudWatchLogGroup:  getEnv("OBSERVABILITY_CLOUDWATCH_LOG_GROUP", ""),
			CloudWatchNamespace: getEnv("OBSERVABILITY_CLOUDWATCH_NAMESPACE", ""),
		},

		RabbitMQ: RabbitMQConfig{
			URL:           getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Queue:         getEnv("RABBITMQ_QUEUE", "default-queue"),
			PrefetchCount: getInt("RABBITMQ_PREFETCH_COUNT", 10),
			Timeout:       getDuration("RABBITMQ_TIMEOUT", "30s"),
		},
	}

	// Apply defaults
	cfg.applyDefaults()

	return cfg, nil
}

// IsLoaded returns whether configuration has been loaded
func (p *Provider) IsLoaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loaded
}
