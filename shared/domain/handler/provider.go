package handler

import (
	"fmt"
	"shared/config"
	"sync"
)

type Provider struct {
	handler     Handler
	adapter     Adapter
	mu          sync.RWMutex
	initialized bool
}

var (
	instance *Provider
	once     sync.Once
)

func GetProvider() *Provider {
	once.Do(func() {
		instance = &Provider{}
	})
	return instance
}

func (p *Provider) Initialize(
	useCase UseCase,
	cfg *config.Config,
	factory HandlerFactory,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	if factory == nil {
		return fmt.Errorf("handler factory is required")
	}

	if useCase == nil {
		return fmt.Errorf("use case is required")
	}

	handler, err := factory.CreateHandler(useCase, cfg)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	// Factory creates adapter
	adapter, err := factory.CreateAdapter(handler, cfg)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	p.handler = handler
	p.adapter = adapter
	p.initialized = true
	return nil
}

func (p *Provider) MustGetHandler() Handler {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		panic("handler not initialized")
	}
	return p.handler
}

func (p *Provider) GetHandler() (Handler, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, fmt.Errorf("handler not initialized")
	}
	return p.handler, nil
}

func (p *Provider) Start() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return fmt.Errorf("handler not initialized")
	}

	return p.adapter.Start()
}
