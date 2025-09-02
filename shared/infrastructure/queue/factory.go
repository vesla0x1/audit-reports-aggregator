package queue

import (
	"fmt"
	"shared/application/ports"
	"shared/infrastructure/config"
)

func CreateQueue(cfg *config.Config, obs ports.Observability) (ports.Queue, error) {
	logger, err := obs.LoggerScoped("queue.factory")
	if err != nil {
		return nil, fmt.Errorf("failed to get logger from observability: %w", err)
	}

	switch cfg.Adapters.Queue {
	case "rabbitmq":
		logger.Info("Creating RabbitMQ queue adapter",
			"url", cfg.Queue.RabbitMQ.URL)
		return NewRabbitMQQueue(&cfg.Queue.RabbitMQ, obs)

	case "sqs":
		logger.Info("Creating SQS queue adapter",
			"region", cfg.Queue.SQS.Region)
		return NewSQSQueue(&cfg.Queue.SQS, obs)

	default:
		return nil, fmt.Errorf("unsupported queue adapter: %s", cfg.Adapters.Queue)
	}
}
