package runtime

import (
	"fmt"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// Create creates the appropriate runtime based on configuration
func Create(cfg *config.Config, handler ports.Handler, obs ports.Observability) (ports.Runtime, error) {
	switch cfg.Adapters.Runtime {
	case "lambda":
		return NewLambdaRuntime(&cfg.Lambda, handler, obs), nil
	case "http":
		return NewHTTPRuntime(&cfg.HTTP, handler, obs), nil
	case "rabbitmq":
		return NewRabbitMQRuntime(&cfg.Queue, handler, obs), nil
	default:
		return nil, fmt.Errorf("unsupported handler adapter: %s", cfg.Adapters.Runtime)
	}
}
