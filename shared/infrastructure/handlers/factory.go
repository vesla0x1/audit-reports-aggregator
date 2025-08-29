package infrahandler

import (
	"shared/config"
	"shared/domain/handler"
	"shared/domain/observability"
	"shared/infrastructure/handlers/adapters/lambda"
)

type Factory struct {
	Logger  observability.Logger
	Metrics observability.Metrics
}

func (f *Factory) CreateHandler(useCase handler.UseCase, cfg *config.Config) (handler.Handler, error) {
	return NewHandler(useCase, cfg, f.Logger, f.Metrics), nil
}

func (f *Factory) CreateAdapter(h handler.Handler, cfg *config.Config) (handler.Adapter, error) {
	// Since you only use Lambda, create Lambda adapter
	return lambda.NewAdapter(h, &cfg.Lambda), nil
}
