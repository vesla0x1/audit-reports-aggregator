package infrahandler

import (
	"shared/config"
	"shared/domain/handler"
	"shared/infrastructure/handlers/adapters/lambda"
)

type Factory struct{}

func (f *Factory) CreateHandler(useCase handler.UseCase, cfg *config.Config) (handler.Handler, error) {
	/*obsProvider := observability.GetProvider()
	logger, err := obsProvider.GetLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to get logger: %w", err)
	}

	metrics, err := obsProvider.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	componentLogger := logger.WithFields(map[string]interface{}{
		"component": "handler",
	})*/

	return NewHandler(useCase, cfg), nil
}

func (f *Factory) CreateAdapter(h handler.Handler, cfg *config.Config) (handler.Adapter, error) {
	// Since you only use Lambda, create Lambda adapter
	return lambda.NewAdapter(h, &cfg.Lambda), nil
}
