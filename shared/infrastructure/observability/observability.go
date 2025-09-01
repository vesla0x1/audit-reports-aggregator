package observability

import (
	"fmt"
	"shared/application/ports"
	"shared/infrastructure/config"
)

type observability struct {
	config  *config.Config
	logger  ports.Logger
	metrics ports.Metrics
}

func CreateObservability(cfg *config.Config) (ports.Observability, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	logger, metrics, err := createObservability(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create observability: %w", err)
	}

	return &observability{
		config:  cfg,
		logger:  logger,
		metrics: metrics,
	}, nil
}

// Components returns logger and metrics without any scoping
// This is typically used when you want to add your own scoping
func (obs *observability) Components() (ports.Logger, ports.Metrics, error) {
	if obs.logger == nil || obs.metrics == nil {
		return nil, nil, fmt.Errorf("observability not initialized")
	}
	return obs.logger, obs.metrics, nil
}

// ComponentsScoped returns logger and metrics scoped to a specific component
func (obs *observability) ComponentsScoped(component string) (ports.Logger, ports.Metrics, error) {
	if obs.logger == nil || obs.metrics == nil {
		return nil, nil, fmt.Errorf("observability not initialized")
	}

	logger := obs.getScopedLogger(component)
	metrics := obs.getScopedMetrics(component)

	return logger, metrics, nil
}

// Logger returns the root logger without scoping
func (obs *observability) Logger() (ports.Logger, error) {
	if obs.logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}
	return obs.logger, nil
}

// LoggerScoped returns a logger scoped to a specific component
func (obs *observability) LoggerScoped(component string) (ports.Logger, error) {
	if obs.logger == nil {
		return nil, fmt.Errorf("logger not initialized")
	}
	return obs.getScopedLogger(component), nil
}

// Metrics returns the root metrics without scoping
func (obs *observability) Metrics() (ports.Metrics, error) {
	if obs.metrics == nil {
		return nil, fmt.Errorf("metrics not initialized")
	}
	return obs.metrics, nil
}

// MetricsScoped returns metrics scoped to a specific component
func (obs *observability) MetricsScoped(component string) (ports.Metrics, error) {
	if obs.metrics == nil {
		return nil, fmt.Errorf("metrics not initialized")
	}
	return obs.getScopedMetrics(component), nil
}

// getScopedLogger creates a logger with component and service context
func (obs *observability) getScopedLogger(component string) ports.Logger {
	return obs.logger.WithFields(map[string]interface{}{
		"service":   obs.config.ServiceName,
		"version":   obs.config.Version,
		"env":       obs.config.Environment,
		"component": component,
	})
}

// getScopedMetrics creates metrics with component and service tags
func (obs *observability) getScopedMetrics(component string) ports.Metrics {
	return obs.metrics.WithTags(map[string]string{
		"service":   obs.config.ServiceName,
		"version":   obs.config.Version,
		"env":       obs.config.Environment,
		"component": component,
	})
}
