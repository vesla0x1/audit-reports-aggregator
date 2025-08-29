package observability

import (
	"errors"
	"fmt"
	"sync"

	"shared/config"
)

// Provider manages observability lifecycle
type Provider struct {
	logger      Logger
	metrics     Metrics
	config      *config.Config
	mu          sync.RWMutex
	initialized bool
}

var (
	instance *Provider
	once     sync.Once
)

// GetProvider returns singleton observability provider
func GetProvider() *Provider {
	once.Do(func() {
		instance = &Provider{}
	})
	return instance
}

// Initialize sets up logger and metrics based on configuration
func (p *Provider) Initialize(cfg *config.Config, factory ObservabilityFactory) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	if factory == nil {
		return errors.New("observability factory is required")
	}

	// Use factory to create observability components
	logger, metrics, err := factory.CreateObservability(cfg)
	if err != nil {
		return fmt.Errorf("failed to create observability: %w", err)
	}

	p.logger = logger
	p.metrics = metrics
	p.config = cfg
	p.initialized = true

	return nil
}

func (p *Provider) GetObservability(component string) (Logger, Metrics, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, nil, errors.New("observability provider not initialized")
	}

	logger := p.getScopedLogger(component)
	metrics := p.getScopedMetrics(component)

	return logger, metrics, nil
}

func (p *Provider) GetLogger(component string) (Logger, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("observability provider not initialized")
	}

	return p.getScopedLogger(component), nil
}

func (p *Provider) GetMetrics(component string) (Metrics, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("observability provider not initialized")
	}
	return p.getScopedMetrics(component), nil
}

// MustGetLogger returns logger with component field
func (p *Provider) MustGetLogger(component string) Logger {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		panic("observability not initialized")
	}

	return p.logger.WithFields(map[string]interface{}{"component": component})
}

// MustGetMetrics returns metrics instance
func (p *Provider) MustGetMetrics(component string) Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics, err := p.GetMetrics(component)
	if err != nil {
		panic(fmt.Sprintf("failed to get metrics: %v", err))
	}
	return metrics
}

// MustGetObservability returns both logger and metrics for a component, panics if not initialized
func (p *Provider) MustGetObservability(component string) (Logger, Metrics) {
	logger, metrics, err := p.GetObservability(component)
	if err != nil {
		panic(fmt.Sprintf("failed to get observability: %v", err))
	}
	return logger, metrics
}

func (p *Provider) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}

// getScopedLogger creates a logger with component and service metadata
// This is an internal helper to avoid code duplication
func (p *Provider) getScopedLogger(component string) Logger {
	// Add standard fields including component
	return p.logger.WithFields(map[string]interface{}{
		"service":   p.config.ServiceName,
		"version":   p.config.Version,
		"env":       p.config.Environment,
		"component": component,
	})
}

// getScopedMetrics creates metrics with component namespace and tags
func (p *Provider) getScopedMetrics(component string) Metrics {
	// Use the Metrics interface's methods for scoping
	// The implementation is in the infrastructure layer
	return p.metrics.WithTags(map[string]string{
		"service":   p.config.ServiceName,
		"version":   p.config.Version,
		"env":       p.config.Environment,
		"component": component,
	})
}
