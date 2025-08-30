package infraobs

import (
	"fmt"

	"shared/config"
	"shared/domain/observability"
	cwAdapter "shared/infrastructure/observability/adapters/cloudwatch"
	stdoutAdapter "shared/infrastructure/observability/adapters/stdout"
)

type Factory struct{}

func NewFactory() observability.ObservabilityFactory {
	return &Factory{}
}

func (f *Factory) Create(cfg *config.Config) (*observability.ObservabilityComponents, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Create logger based on adapter configuration
	logger, err := f.createLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create metrics based on adapter configuration
	metrics, err := f.createMetrics(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	return &observability.ObservabilityComponents{
		Logger:  logger,
		Metrics: metrics,
	}, nil
}

func (f *Factory) createLogger(cfg *config.Config) (observability.Logger, error) {
	switch cfg.Adapters.Logger {
	case "cloudwatch":
		return cwAdapter.NewLogger(*cfg)

	case "stdout":
		return stdoutAdapter.NewLogger(), nil

	default:
		return nil, fmt.Errorf("unsupported logger adapter: %s", cfg.Adapters.Logger)
	}
}

func (f *Factory) createMetrics(cfg *config.Config) (observability.Metrics, error) {
	switch cfg.Adapters.Metrics {
	case "cloudwatch":
		return cwAdapter.NewMetrics(*cfg)

	case "stdout":
		return stdoutAdapter.NewMetrics(), nil

	default:
		return nil, fmt.Errorf("unsupported metrics adapter: %s", cfg.Adapters.Metrics)
	}
}
