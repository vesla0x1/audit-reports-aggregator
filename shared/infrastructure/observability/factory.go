package observability

import (
	"fmt"

	"shared/application/ports"
	"shared/infrastructure/config"
	"shared/infrastructure/observability/cloudwatch"
	"shared/infrastructure/observability/stdout"
)

func createObservability(cfg *config.Config) (ports.Logger, ports.Metrics, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("configuration is required")
	}

	// Create logger based on adapter configuration
	logger, err := createLogger(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create metrics based on adapter configuration
	metrics, err := createMetrics(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	return logger, metrics, nil
}

func createLogger(cfg *config.Config) (ports.Logger, error) {
	switch cfg.Adapters.Logger {
	case "cloudwatch":
		return cloudwatch.NewCloudwatchLogger(*cfg)

	case "stdout":
		return stdout.NewStdoutLogger()

	default:
		return nil, fmt.Errorf("unsupported logger adapter: %s", cfg.Adapters.Logger)
	}
}

func createMetrics(cfg *config.Config) (ports.Metrics, error) {
	switch cfg.Adapters.Metrics {
	case "cloudwatch":
		return cloudwatch.NewCloudwatchMetrics(*cfg)

	case "stdout":
		return stdout.NewStdoutMetrics()

	default:
		return nil, fmt.Errorf("unsupported metrics adapter: %s", cfg.Adapters.Metrics)
	}
}
