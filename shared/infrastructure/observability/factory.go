package infraobs

import (
	"fmt"
	"shared/config"
	"shared/domain/observability"
	cwAdapter "shared/infrastructure/observability/adapters/cloudwatch"
)

type Factory struct{}

func (f *Factory) Create(cfg *config.Config) (*observability.ObservabilityComponents, error) {
	// Validate config
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	// Create CloudWatch logger
	logger, err := cwAdapter.NewLogger(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch logger: %w", err)
	}

	// Create CloudWatch metrics
	metrics, err := cwAdapter.NewMetrics(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch metrics: %w", err)
	}

	return &observability.ObservabilityComponents{
		Logger:  logger,
		Metrics: metrics,
	}, nil
}
