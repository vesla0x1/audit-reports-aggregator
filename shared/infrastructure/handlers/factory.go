package infrahandler

import (
	"fmt"
	"shared/config"
	"shared/domain/handler"
	"shared/domain/observability"
	"shared/infrastructure/handlers/adapters/lambda"
)

type Factory struct {
	useCase handler.UseCase
	Logger  observability.Logger
	Metrics observability.Metrics
}

func (f *Factory) Create(cfg *config.Config) (*handler.HandlerComponents, error) {
	if f.useCase == nil {
		return nil, fmt.Errorf("use case is required")
	}

	h := NewHandler(f.useCase, cfg, f.Logger, f.Metrics)
	a := lambda.NewAdapter(h, &cfg.Lambda)

	return &handler.HandlerComponents{
		Handler: h,
		Adapter: a,
	}, nil
}

func (f *Factory) SetUseCase(useCase handler.UseCase) {
	f.useCase = useCase
}
