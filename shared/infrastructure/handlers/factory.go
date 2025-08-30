package infrahandler

import (
	"fmt"

	"shared/config"
	"shared/domain/handler"
	"shared/domain/observability"
	httpAdapter "shared/infrastructure/handlers/adapters/http"
	lambdaAdapter "shared/infrastructure/handlers/adapters/lambda"
	rabbitAdapter "shared/infrastructure/handlers/adapters/rabbitmq"
)

type Factory struct {
	useCase handler.UseCase
	logger  observability.Logger
	metrics observability.Metrics
}

func NewFactory(logger observability.Logger, metrics observability.Metrics) *Factory {
	return &Factory{
		logger:  logger,
		metrics: metrics,
	}
}

func (f *Factory) SetUseCase(useCase handler.UseCase) {
	f.useCase = useCase
}

func (f *Factory) Create(cfg *config.Config) (*handler.HandlerComponents, error) {
	if f.useCase == nil {
		return nil, fmt.Errorf("use case is required")
	}

	// Create handler
	h := NewHandler(f.useCase, cfg, f.logger, f.metrics)

	// Create adapter based on configuration
	adapter, err := f.createAdapter(h, cfg)
	if err != nil {
		return nil, err
	}

	return &handler.HandlerComponents{
		Handler: h,
		Adapter: adapter,
	}, nil
}

func (f *Factory) createAdapter(h handler.Handler, cfg *config.Config) (handler.Adapter, error) {
	switch cfg.Adapters.Handler {
	case "lambda":
		f.logger.Info("Creating Lambda handler adapter")
		return lambdaAdapter.NewAdapter(h, &cfg.Lambda), nil

	case "http":
		f.logger.Info("Creating HTTP handler adapter", "address", cfg.HTTP.Addr)
		return httpAdapter.NewAdapter(h, &cfg.HTTP), nil

	case "rabbitmq":
		f.logger.Info("Creating RabbitMQ handler adapter", "queue", cfg.RabbitMQ.Queue)
		return rabbitAdapter.NewAdapter(h, &cfg.RabbitMQ), nil

	default:
		return nil, fmt.Errorf("unsupported handler adapter: %s", cfg.Adapters.Handler)
	}
}
