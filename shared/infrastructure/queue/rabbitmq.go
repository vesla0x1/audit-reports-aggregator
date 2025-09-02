package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shared/application/ports"
	"shared/infrastructure/config"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQQueue struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	logger  ports.Logger
	metrics ports.Metrics
	config  *config.RabbitMQConfig
}

func NewRabbitMQQueue(cfg *config.RabbitMQConfig, obs ports.Observability) (ports.Queue, error) {
	logger, metrics, err := obs.ComponentsScoped("queue.rabbitmq")
	if err != nil {
		return nil, fmt.Errorf("failed to get observability components: %w", err)
	}

	// Connect to RabbitMQ
	conn, err := amqp091.Dial(cfg.URL)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", "error", err)
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		logger.Error("failed to create channel", "error", err)
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	logger.Info("RabbitMQ queue initialized successfully")

	return &RabbitMQQueue{
		conn:    conn,
		channel: channel,
		logger:  logger,
		metrics: metrics,
		config:  cfg,
	}, nil
}

func (q *RabbitMQQueue) Publish(ctx context.Context, message *ports.QueueMessage) error {
	startTime := time.Now()
	defer func() {
		q.metrics.RecordHistogram("queue.publish.duration",
			time.Since(startTime).Seconds(),
			map[string]string{"target": message.Target})
	}()

	// Marshal message body to JSON
	body, err := json.Marshal(message.Body)
	if err != nil {
		q.logger.Error("failed to marshal message", "error", err)
		q.metrics.IncrementCounter("queue.publish.error",
			map[string]string{"target": message.Target, "error": "marshal_failed"})
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Declare queue (idempotent operation)
	_, err = q.channel.QueueDeclare(
		message.Target, // queue name
		true,           // durable
		false,          // auto-delete
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		q.logger.Error("failed to declare queue", "error", err, "queue", message.Target)
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Create AMQP message
	amqpMsg := amqp091.Publishing{
		DeliveryMode: amqp091.Persistent,
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now(),
	}

	// Publish message
	err = q.channel.PublishWithContext(
		ctx,
		"",             // exchange (empty for direct queue)
		message.Target, // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqpMsg,
	)

	if err != nil {
		q.logger.Error("failed to publish message", "error", err, "target", message.Target)
		q.metrics.IncrementCounter("queue.publish.error",
			map[string]string{"target": message.Target, "error": "publish_failed"})
		return fmt.Errorf("failed to publish message: %w", err)
	}

	q.logger.Info("message published successfully", "target", message.Target, "size", len(body))
	q.metrics.IncrementCounter("queue.publish.success",
		map[string]string{"target": message.Target})

	return nil
}

func (q *RabbitMQQueue) PublishBatch(ctx context.Context, messages []*ports.QueueMessage) error {
	for _, msg := range messages {
		if err := q.Publish(ctx, msg); err != nil {
			return fmt.Errorf("failed to publish message in batch: %w", err)
		}
	}
	return nil
}

func (q *RabbitMQQueue) Close() error {
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}
