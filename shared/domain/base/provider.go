package base

import (
	"fmt"
	"shared/config"
)

// Provider manages a singleton resource
type Provider[T any] struct {
	resource    T
	config      *config.Config
	initialized bool
}

var (
	registry = make(map[string]any)
)

// GetProvider returns a singleton provider instance for the given key
func GetProvider[T any](key string) *Provider[T] {
	if p, exists := registry[key]; exists {
		if provider, ok := p.(*Provider[T]); ok {
			return provider
		}
		panic(fmt.Sprintf("provider type mismatch for key %s", key))
	}

	p := &Provider[T]{}
	registry[key] = p
	return p
}

// Initialize initializes the provider with config and factory
func (p *Provider[T]) Initialize(cfg *config.Config, factory Factory[T]) error {
	if p.initialized {
		return nil
	}

	if factory == nil {
		return fmt.Errorf("factory is required")
	}

	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	resource, err := factory.Create(cfg)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	p.resource = resource
	p.config = cfg
	p.initialized = true
	return nil
}

// Get returns the resource if initialized
func (p *Provider[T]) Get() (T, error) {
	var zero T
	if !p.initialized {
		return zero, fmt.Errorf("provider not initialized")
	}
	return p.resource, nil
}

// MustGet returns the resource or panics if not initialized
func (p *Provider[T]) MustGet() T {
	resource, err := p.Get()
	if err != nil {
		panic(fmt.Sprintf("failed to get resource: %v", err))
	}
	return resource
}

// GetConfig returns the provider's configuration
func (p *Provider[T]) GetConfig() (*config.Config, error) {
	if !p.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}
	return p.config, nil
}

// IsInitialized returns whether the provider has been initialized
func (p *Provider[T]) IsInitialized() bool {
	return p.initialized
}

// Reset resets the provider state (useful for testing)
func (p *Provider[T]) Reset() {
	var zero T
	p.resource = zero
	p.config = nil
	p.initialized = false
}
