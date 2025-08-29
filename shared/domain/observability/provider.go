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

func (p *Provider) GetObservability() (Logger, Metrics, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, nil, errors.New("observability provider not initialized")
	}
	return p.logger, p.metrics, nil
}

func (p *Provider) GetLogger() (Logger, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("observability provider not initialized")
	}
	return p.logger, nil
}

func (p *Provider) GetMetrics() (Metrics, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("observability provider not initialized")
	}
	return p.metrics, nil
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
func (p *Provider) MustGetMetrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		panic("observability not initialized")
	}

	return p.metrics
}

func (p *Provider) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.initialized
}
