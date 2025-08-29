package handler

import (
	"context"
	"shared/config"
	"shared/domain/observability"
)

// Handler processes requests through middleware chains
type Handler interface {
	Handle(ctx context.Context, req Request) (Response, error)
	Logger() observability.Logger
	Metrics() observability.Metrics
}

// UseCase defines business logic interface (renamed from Worker)
type UseCase interface {
	Name() string
	Run(ctx context.Context, req Request) (Response, error)
}

// Middleware wraps handler functions
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc processes a single request
type HandlerFunc func(ctx context.Context, req Request) (Response, error)

// Adapter interface for platform-specific adapters
type Adapter interface {
	Start() error
}

type HandlerFactory interface {
	CreateHandler(useCase UseCase, cfg *config.Config) (Handler, error)
	CreateAdapter(handler Handler, cfg *config.Config) (Adapter, error)
}
