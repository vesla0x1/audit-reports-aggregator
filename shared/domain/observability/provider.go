package observability

import (
	"fmt"
	"shared/config"
	"shared/domain/base"
)

// Public API using generic provider
const observabilityProviderKey = "observability"

type ObservabilityComponents struct {
	Logger  Logger
	Metrics Metrics
}

func GetProvider() *base.Provider[*ObservabilityComponents] {
	return base.GetProvider[*ObservabilityComponents](observabilityProviderKey)
}

func Initialize(cfg *config.Config, factory ObservabilityFactory) error {
	return GetProvider().Initialize(cfg, factory)
}

func GetObservability(component string) (Logger, Metrics, error) {
	components, err := GetProvider().Get()
	if err != nil {
		return nil, nil, err
	}

	cfg, err := GetProvider().GetConfig()
	if err != nil {
		return nil, nil, err
	}

	logger := getScopedLogger(components.Logger, cfg, component)
	metrics := getScopedMetrics(components.Metrics, cfg, component)

	return logger, metrics, nil
}

func GetLogger(component string) (Logger, error) {
	components, err := GetProvider().Get()
	if err != nil {
		return nil, err
	}

	cfg, err := GetProvider().GetConfig()
	if err != nil {
		return nil, err
	}

	return getScopedLogger(components.Logger, cfg, component), nil
}

func GetMetrics(component string) (Metrics, error) {
	components, err := GetProvider().Get()
	if err != nil {
		return nil, err
	}

	cfg, err := GetProvider().GetConfig()
	if err != nil {
		return nil, err
	}

	return getScopedMetrics(components.Metrics, cfg, component), nil
}

func MustGetObservability(component string) (Logger, Metrics) {
	logger, metrics, err := GetObservability(component)
	if err != nil {
		panic(fmt.Sprintf("failed to get observability: %v", err))
	}
	return logger, metrics
}

func MustGetLogger(component string) Logger {
	logger, err := GetLogger(component)
	if err != nil {
		panic(fmt.Sprintf("failed to get logger: %v", err))
	}
	return logger
}

func MustGetMetrics(component string) Metrics {
	metrics, err := GetMetrics(component)
	if err != nil {
		panic(fmt.Sprintf("failed to get metrics: %v", err))
	}
	return metrics
}

func IsInitialized() bool {
	return GetProvider().IsInitialized()
}

func Reset() {
	GetProvider().Reset()
}

// Helper functions for scoping
func getScopedLogger(logger Logger, cfg *config.Config, component string) Logger {
	return logger.WithFields(map[string]interface{}{
		"service":   cfg.ServiceName,
		"version":   cfg.Version,
		"env":       cfg.Environment,
		"component": component,
	})
}

func getScopedMetrics(metrics Metrics, cfg *config.Config, component string) Metrics {
	return metrics.WithTags(map[string]string{
		"service":   cfg.ServiceName,
		"version":   cfg.Version,
		"env":       cfg.Environment,
		"component": component,
	})
}
