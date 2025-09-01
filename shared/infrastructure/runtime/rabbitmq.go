package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"

	"shared/application/ports"
	"shared/infrastructure/config"
)

// handles RabbitMQ consumer runtime integration
type rabbitmqRuntime struct {
	handler ports.Handler
	logger  ports.Logger
	metrics ports.Metrics
	config  *config.RabbitMQConfig
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQRuntime creates a new RabbitMQ runtime
func NewRabbitMQRuntime(cfg *config.RabbitMQConfig, handler ports.Handler, obs ports.Observability) ports.Runtime {
	logger, metrics, err := obs.ComponentsScoped("runtime.rabbitmq")
	if err != nil {
		panic(fmt.Errorf("failed to create runtime: Observability was not initialized %w", err))
	}

	if handler == nil {
		panic(fmt.Errorf("failed to create runtime: handler is required"))
	}

	return &rabbitmqRuntime{
		handler: handler,
		logger:  logger,
		metrics: metrics,
		config:  cfg,
	}
}

// Start begins consuming messages from RabbitMQ
func (runtime *rabbitmqRuntime) Start() error {
	// Connect
	conn, err := amqp.Dial(runtime.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	runtime.conn = conn

	// Open channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}
	runtime.channel = ch

	// Set QoS
	if runtime.config.PrefetchCount > 0 {
		if err := ch.Qos(runtime.config.PrefetchCount, 0, false); err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("failed to set QoS: %w", err)
		}
	}

	// Declare queue (idempotent - creates if doesn't exist)
	q, err := ch.QueueDeclare(
		runtime.config.Queue, // name
		true,                 // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer tag (auto-generated)
		false,  // auto-ack (we'll ack manually)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to consume: %w", err)
	}

	runtime.logger.Info("RabbitMQ consumer started",
		"queue", runtime.config.Queue,
		"prefetch", runtime.config.PrefetchCount)
	runtime.metrics.IncrementCounter("rabbitmq.starts", nil)

	// Process messages
	for msg := range msgs {
		runtime.processMessage(msg)
	}

	return nil
}

// processMessage handles a single message
func (runtime *rabbitmqRuntime) processMessage(msg amqp.Delivery) {
	startTime := time.Now()

	// Create context with timeout
	ctx := context.Background()
	if runtime.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, runtime.config.Timeout)
		defer cancel()
	}

	// Convert to ports.RuntimeRequest
	req := ports.RuntimeRequest{
		ID:        msg.MessageId,
		Source:    "rabbitmq",
		Type:      runtime.extractType(msg),
		Payload:   json.RawMessage(msg.Body),
		Metadata:  runtime.buildMetadata(msg),
		Timestamp: msg.Timestamp,
	}

	// Set defaults
	if req.ID == "" {
		req.ID = fmt.Sprintf("rmq-%d", msg.DeliveryTag)
	}
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// Log request
	runtime.logger.Info("Processing RabbitMQ message",
		"message_id", req.ID,
		"routing_key", msg.RoutingKey,
		"redelivered", msg.Redelivered)
	runtime.metrics.IncrementCounter("rabbitmq.messages", nil)

	// Process
	resp, err := runtime.handler.Handle(ctx, req)

	// Handle result
	if err == nil && resp.Success {
		// Success - acknowledge
		if err := msg.Ack(false); err != nil {
			runtime.logger.Error("Failed to ack message",
				"id", req.ID,
				"error", err)
		}
		runtime.logger.Info("Message processed successfully",
			"id", req.ID,
			"duration_ms", time.Since(startTime).Milliseconds())
		runtime.metrics.IncrementCounter("rabbitmq.success", nil)
	} else {
		// Failure - reject and requeue if not already redelivered
		requeue := !msg.Redelivered
		if err := msg.Nack(false, requeue); err != nil {
			runtime.logger.Error("Failed to nack message",
				"id", req.ID,
				"error", err)
		}

		if err != nil {
			runtime.logger.Error("Message processing failed",
				"id", req.ID,
				"error", err,
				"requeued", requeue)
		} else if !resp.Success {
			runtime.logger.Error("Message processing unsuccessful",
				"id", req.ID,
				"error", resp.Error,
				"requeued", requeue)
		}
		runtime.metrics.IncrementCounter("rabbitmq.failure", nil)
	}

	// Record duration
	runtime.metrics.RecordHistogram("rabbitmq.duration_ms",
		float64(time.Since(startTime).Milliseconds()), nil)
}

// Stop gracefully shuts down the consumer
func (runtime *rabbitmqRuntime) Stop(ctx context.Context) error {
	if runtime.channel != nil {
		runtime.channel.Close()
	}
	if runtime.conn != nil {
		runtime.conn.Close()
	}
	runtime.logger.Info("RabbitMQ consumer stopped")
	return nil
}

// extractType gets message type from headers or routing key
func (runtime *rabbitmqRuntime) extractType(msg amqp.Delivery) string {
	// Check headers first
	if t, ok := msg.Headers["type"]; ok {
		return fmt.Sprintf("%v", t)
	}
	// Use routing key if available
	if msg.RoutingKey != "" {
		return msg.RoutingKey
	}
	return "message"
}

// buildMetadata creates metadata from message properties
func (runtime *rabbitmqRuntime) buildMetadata(msg amqp.Delivery) map[string]string {
	meta := make(map[string]string)

	if msg.RoutingKey != "" {
		meta["routing_key"] = msg.RoutingKey
	}
	if msg.Exchange != "" {
		meta["exchange"] = msg.Exchange
	}
	if msg.CorrelationId != "" {
		meta["correlation_id"] = msg.CorrelationId
	}
	if msg.ReplyTo != "" {
		meta["reply_to"] = msg.ReplyTo
	}
	meta["redelivered"] = fmt.Sprintf("%v", msg.Redelivered)

	// Add custom headers
	for k, v := range msg.Headers {
		meta[fmt.Sprintf("header_%s", k)] = fmt.Sprintf("%v", v)
	}

	return meta
}
