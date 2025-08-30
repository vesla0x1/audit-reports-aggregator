package handler

import (
	"fmt"
	"shared/config"
	"shared/domain/base"
)

// Public API using generic provider
const handlerProviderKey = "handler"

type HandlerComponents struct {
	Handler Handler
	Adapter Adapter
}

func GetProvider() *base.Provider[*HandlerComponents] {
	return base.GetProvider[*HandlerComponents](handlerProviderKey)
}

// Initialize with UseCase parameter (special case for handler)
func Initialize(useCase UseCase, cfg *config.Config, factory HandlerFactory) error {
	if useCase == nil {
		return fmt.Errorf("use case is required")
	}

	if factory == nil {
		return fmt.Errorf("handler factory is required")
	}

	// Set the use case on the factory
	factory.SetUseCase(useCase)

	return GetProvider().Initialize(cfg, factory)
}

func GetHandler() (Handler, error) {
	components, err := GetProvider().Get()
	if err != nil {
		return nil, err
	}
	return components.Handler, nil
}

func MustGetHandler() Handler {
	handler, err := GetHandler()
	if err != nil {
		panic(fmt.Sprintf("failed to get handler: %v", err))
	}
	return handler
}

func GetAdapter() (Adapter, error) {
	components, err := GetProvider().Get()
	if err != nil {
		return nil, err
	}
	return components.Adapter, nil
}

func MustGetAdapter() Adapter {
	adapter, err := GetAdapter()
	if err != nil {
		panic(fmt.Sprintf("failed to get adapter: %v", err))
	}
	return adapter
}

func Start() error {
	adapter, err := GetAdapter()
	if err != nil {
		return fmt.Errorf("handler not initialized")
	}
	return adapter.Start()
}

func IsInitialized() bool {
	return GetProvider().IsInitialized()
}

func Reset() {
	GetProvider().Reset()
}
